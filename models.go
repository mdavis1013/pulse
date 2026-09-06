package main

// Header is the fixed 12-byte block at the start of every DNS message.
type Header struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

// Question is one entry in the question section of a DNS message —
// what a query is actually asking about.
type Question struct {
	Name  string
	Type  uint16
	Class uint16
}

// ResourceRecord is one answer record in a DNS message — the actual
// data a device sends back.
type ResourceRecord struct {
	Name  string
	Type  uint16
	Class uint16
	TTL   uint32
	Data  []byte

	rdataOffset int
}

// Message is a fully parsed DNS/mDNS packet — a header plus whatever
// questions and answers it contains.
type Message struct {
	Header    Header
	Questions []Question
	Answers   []ResourceRecord

	Raw []byte
}

const (
	TypeA   = 1
	TypePTR = 12
	TypeTXT = 16
	TypeSRV = 33
)