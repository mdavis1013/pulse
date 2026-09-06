# pulse

A from-scratch mDNS network device scanner, written in Go — discovers
real devices on your local network (phones, smart TVs, printers,
speakers, your own Mac) by hand-decoding the DNS wire protocol, not by
calling a library that already does it for you.

## What it does

Every device on your WiFi that supports AirPlay, Chromecast, network
printing, HomeKit, or similar already broadcasts its presence
constantly. Pulse sends the same kind of "please announce yourselves"
query real tools like `dns-sd` use, then decodes every reply itself,
byte by byte, and shows a live, continuously-updating list of what's
actually on the network — including tracking when a device goes quiet
and is no longer considered present.

## Why this is a different problem than a packet sniffer

A tool that just captures and displays packets treats each one
independently. Pulse doesn't: a single device's presence is spread
across three separate records that reference each other — a PTR
record says "an instance exists," an SRV record (found by matching the
PTR's target name) says "here's its host and port," and an A record
(found by matching the SRV's target) says "here's that host's actual
IP." Producing one clean answer means correctly cross-referencing all
three as they arrive in whatever order.

On top of that, DNS names use a real compression trick: instead of
repeating a long name in every record, a name can end in a pointer
meaning "go re-read the rest starting at byte N of this same packet."
`dns.go`'s `decodeName` implements this directly.

## The distributed-systems piece: inferring who's still there

mDNS has no central directory — nothing ever sends an explicit "I'm
leaving" message when a device disconnects. The only signal is
silence, and every record comes with a TTL: "trust this for N
seconds." `registry.go` tracks each device's own declared TTL and
periodically sweeps for anything that's gone silent past it — the
same fundamental pattern as failure detectors in real distributed
systems.

## Real bugs found building this

- **Reverse-DNS PTR records** (`244.1.168.192.in-addr.arpa`) travel as
  the same record type as real service-discovery PTRs, but mean
  something completely different (an IP-to-hostname lookup, not a
  service announcement). Without filtering these out, they showed up
  as nonsense "devices." Fixed in `main.go`'s `isReverseDNSZone`.
- **AirPlay-audio (RAOP) names** are prefixed with a device's MAC
  address (`AA8F74BD69AF@Maria's MacBook Air`), which broke the
  device-grouping logic — it saw that as a different device than plain
  `Maria's MacBook Air`. Fixed in `topology.go`'s `groupKey`.

## How it works

- **`models.go`** — the data shapes: DNS header, question, resource
  record, and a full parsed message.
- **`dns.go`** — hand-rolled DNS wire-format parsing and encoding:
  name (de)compression, resource records, and typed decoders for PTR,
  SRV, and A records.
- **`mdns.go`** — joins the mDNS multicast group (`224.0.0.251:5353`),
  sends queries, and listens continuously for replies.
- **`registry.go`** — the liveness/failure-detection logic: tracks
  each device's declared TTL and infers departure from silence past
  it.
- **`appstate.go`** — thread-safe shared state bridging the background
  discovery loop into the TUI's render loop.
- **`topology.go`** — groups services by inferred physical device for
  the network map view.
- **`export.go`** — writes the current session to a readable JSON
  file.
- **`tui.go`** — the interactive terminal interface (`bubbletea` +
  `lipgloss`): a live device table, a join/leave event feed, a per-
  record detail view with live TTL countdowns, and a network map.
- **`main.go`** — ties it all together: discovers service types,
  follows up on each one, cross-references PTR/SRV/A records as they
  arrive, and runs the interactive program.

## Running it

```
go build -o pulse .
sudo ./pulse
```

(`sudo` is needed the same way Wireshark/tcpdump-style tools need it —
joining a multicast group isn't something an unprivileged socket can
do.)

## Controls

| Key | Action |
|---|---|
| `↑` `↓` | Move through the device list |
| `enter` | Inspect the selected device (PTR/SRV/A detail + live TTL bars) |
| `esc` | Close the detail view |
| `tab` | Toggle the network map (devices grouped by physical device) |
| `e` | Export the current session to a JSON file |
| `q` | Quit |

## Tech stack

| Concern | Choice |
|---|---|
| Language | Go |
| Core logic | Standard library only — DNS parsing, networking, and liveness detection have zero external dependencies |
| TUI | `bubbletea` + `bubbles` + `lipgloss` |
| Protocol | Real DNS wire format over mDNS multicast — hand-decoded |

## Project status & roadmap

Discovery, liveness tracking, the interactive TUI, device grouping,
and export are all complete and working against a real home network.
Not yet built:

- TXT record decoding (often carries useful metadata, e.g. a
  printer's supported paper sizes)
- IPv6 (AAAA records) — currently only IPv4 is decoded
- Automated tests (currently verified by hand against a real network)

Built by Maria Davis · Go · bubbletea · lipgloss