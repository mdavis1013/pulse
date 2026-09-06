package main

import (
	"encoding/binary"
	"fmt"
	"strings"
)
// decodeName reads a DNS name starting at pos in data, and returns the
// decoded name plus the position right after it.
//
// DNS names are stored as repeated [length byte][that many letters],
// ending in a single 0 byte. Example: "_googlecast._tcp.local" is
// stored as: [11]_googlecast [4]_tcp [5]local [0]
//
// A name can also contain a COMPRESSION POINTER instead of more
// letters: an instruction meaning "go re-read the rest of this name
// starting at byte N of this same packet," so a repeated name doesn't
// need to be spelled out twice. A pointer is recognized because its
// length byte has its top two bits both set to 1 -- a real length
// never needs those bits, since a label can never be longer than 63
// letters.
func decodeName(data []byte, pos int) (string, int, error) {
	var labels []string
	endPos := -1

	for {
		if pos >= len(data) {
			return "", 0, fmt.Errorf("position %d is past the end of the packet (length %d)", pos, len(data))
		}

		length := int(data[pos])

		if length == 0 {
			pos++
			if endPos == -1 {
				endPos = pos
			}
			break
		}

		if length&0xC0 == 0xC0 {
			if pos+1 >= len(data) {
				return "", 0, fmt.Errorf("truncated compression pointer at position %d", pos)
			}
			secondByte := int(data[pos+1])
			pointerTarget := (length&0x3F)<<8 | secondByte

			if endPos == -1 {
				endPos = pos + 2
			}
			pos = pointerTarget
			continue
		}

		if pos+1+length > len(data) {
			return "", 0, fmt.Errorf("label at position %d runs past the end of the packet", pos)
		}

		pos++
		label := string(data[pos : pos+length])
		labels = append(labels, label)
		pos += length
	}

	name := strings.Join(labels, ".")
	return name, endPos, nil
}


// decodeRR reads one full resource record starting at pos: its name,
// then type/class/ttl/data-length (each a fixed number of bytes), then
// the actual data.
func decodeRR(data []byte, pos int) (ResourceRecord, int, error) {
	name, pos, err := decodeName(data, pos)
	if err != nil {
		return ResourceRecord{}, pos, err
	}

	rr := ResourceRecord{
		Name:  name,
		Type:  binary.BigEndian.Uint16(data[pos : pos+2]),
		Class: binary.BigEndian.Uint16(data[pos+2 : pos+4]),
		TTL:   binary.BigEndian.Uint32(data[pos+4 : pos+8]),
	}
	pos += 8

	dataLength := int(binary.BigEndian.Uint16(data[pos : pos+2]))
	pos += 2

	rr.rdataOffset = pos
	rr.Data = data[pos : pos+dataLength]
	pos += dataLength

	return rr, pos, nil
}

// ParseMessage decodes an entire raw DNS/mDNS packet into a Message.
func ParseMessage(data []byte) (*Message, error) {
	msg := &Message{
		Header: Header{
			ID:      binary.BigEndian.Uint16(data[0:2]),
			Flags:   binary.BigEndian.Uint16(data[2:4]),
			QDCount: binary.BigEndian.Uint16(data[4:6]),
			ANCount: binary.BigEndian.Uint16(data[6:8]),
			NSCount: binary.BigEndian.Uint16(data[8:10]),
			ARCount: binary.BigEndian.Uint16(data[10:12]),
		},
		Raw: data,
	}

	pos := 12 // the header is always exactly 12 bytes, so questions start right after it

	for i := 0; i < int(msg.Header.QDCount); i++ {
		name, newPos, err := decodeName(data, pos)
		if err != nil {
			return nil, err
		}
		pos = newPos

		q := Question{
			Name:  name,
			Type:  binary.BigEndian.Uint16(data[pos : pos+2]),
			Class: binary.BigEndian.Uint16(data[pos+2 : pos+4]),
		}
		pos += 4

		msg.Questions = append(msg.Questions, q)
	}

	totalRecords := int(msg.Header.ANCount) + int(msg.Header.NSCount) + int(msg.Header.ARCount)
	for i := 0; i < totalRecords; i++ {
		rr, newPos, err := decodeRR(data, pos)
		if err != nil {
			return nil, err
		}
		pos = newPos

		msg.Answers = append(msg.Answers, rr)
	}

	return msg, nil
}

// EncodeQuery builds the raw bytes for a DNS/mDNS query asking about
// the given name -- this is the actual "shout into the room" message.
func EncodeQuery(name string) []byte {
	buf := make([]byte, 12) // start with an empty 12-byte header

	// QDCount = 1 -- "this message contains exactly one question"
	buf[4] = 0x00
	buf[5] = 0x01

	buf = append(buf, encodeName(name)...)

	buf = append(buf, 0x00, 0x0c) // Type = 12 (PTR)
	buf = append(buf, 0x00, 0x01) // Class = 1 (IN)

	return buf
}

// encodeName writes a name as the length-prefixed labels DNS expects --
// the exact reverse of decodeName, with no compression (we're only
// ever encoding short queries, so there's nothing worth compressing).
func encodeName(name string) []byte {
	var buf []byte
	for _, label := range strings.Split(name, ".") {
		buf = append(buf, byte(len(label)))
		buf = append(buf, label...)
	}
	buf = append(buf, 0)
	return buf
}

// SRVData is a decoded SRV record: which host and port actually
// provides a service.
type SRVData struct {
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   string
}

// DecodeSRV interprets a SRV record's raw Data as priority, weight,
// port, plus a (possibly compressed) target hostname.
func (rr ResourceRecord) DecodeSRV(fullPacket []byte) (SRVData, error) {
	if len(rr.Data) < 6 {
		return SRVData{}, fmt.Errorf("SRV record too short")
	}

	target, _, err := decodeName(fullPacket, rr.rdataOffset+6)
	if err != nil {
		return SRVData{}, err
	}

	return SRVData{
		Priority: binary.BigEndian.Uint16(rr.Data[0:2]),
		Weight:   binary.BigEndian.Uint16(rr.Data[2:4]),
		Port:     binary.BigEndian.Uint16(rr.Data[4:6]),
		Target:   target,
	}, nil
}

// DecodeA interprets an A record's raw Data as an IPv4 address.
func (rr ResourceRecord) DecodeA() (string, error) {
	if len(rr.Data) != 4 {
		return "", fmt.Errorf("A record is not 4 bytes")
	}
	return fmt.Sprintf("%d.%d.%d.%d", rr.Data[0], rr.Data[1], rr.Data[2], rr.Data[3]), nil
}