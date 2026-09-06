 ____  _    _ _     ____  ______
|  _ \| |  | | |   / ___||  ____|
| |_) | |  | | |   \___ \| |__
|  __/| |  | | |    ___) |  __|
| |   | |__| | |___|____/| |____
|_|    \____/|______|

A live mDNS network device scanner with a terminal UI — discovers real
devices on your WiFi by hand-decoding the DNS wire protocol, built
from scratch in Go.

Go Multicast TUI

## What it does

Every device on your WiFi that supports AirPlay, Chromecast, network
printing, HomeKit, or similar already broadcasts its presence
constantly. Pulse sends the same kind of "please announce yourselves"
query real tools like `dns-sd` use, then decodes every reply itself —
byte by byte, no DNS library involved — and shows a live,
continuously-updating view of what's actually on the network: which
devices exist, where to reach them, and whether they're still there.

## Highlights

- **Real mDNS multicast discovery** (`224.0.0.251:5353`) — no
  simulated data; every device shown is something actually announcing
  itself on the network at that moment.
- **Hand-rolled DNS wire-format parsing** — including name
  compression, where a record can say "the rest of this name is
  identical to bytes already written earlier in this packet" instead
  of repeating it.
- **Three-record reconstruction** — a device's identity is spread
  across a PTR, an SRV, and an A record, which can arrive
  independently and in any order. Pulse cross-references all three
  into one resolved entry.
- **TTL-based liveness detection** — nothing in mDNS ever announces
  "I'm leaving." Each device's own declared TTL drives a background
  sweep that infers departure from silence — the same core pattern
  real distributed systems use for failure detection.
- **Device grouping** — independent service announcements that share
  an underlying device (including AirPlay-audio's MAC-prefixed naming
  quirk) are recognized and grouped as one physical device, not several.
- **Live interactive TUI** — a device table, a join/leave event feed,
  a per-record detail view with live-draining TTL bars, and a
  toggleable network map.
- **Session export** — dump the current device list and event history
  to a readable JSON file.

## How it works

Pulse is organized as four cooperating layers, all in one Go binary:

```
┌───────────────────────────────────────────────────────────┐
│                  Bubble Tea TUI (model)                     │
│                                                             │
│   ┌───────────┐    ┌────────────┐    ┌─────────────────┐    │
│   │  device   │ ↔  │   detail   │ ↔  │   network map    │    │
│   │  table    │    │   view     │    │   (grouped)      │    │
│   └───────────┘    └────────────┘    └─────────────────┘    │
│         │                                                   │
│         ▼                                                   │
│   ┌───────────────────────────────┐                         │
│   │  Registry + AppState           │  ← source of truth       │
│   │  (TTL-based liveness,          │                         │
│   │   event log)                   │                         │
│   └───────────────▲───────────────┘                         │
└───────────────────┼─────────────────────────────────────────┘
                     │  Observe() / AddEvent() / refresh signal
        ┌────────────┴─────────────┐
        │   Discovery loop          │
        │   cross-references        │
        │   PTR / SRV / A records   │
        └────────────▲─────────────┘
                     │  parsed *Message
        ┌────────────┴─────────────┐
        │   Listen goroutine        │
        │   mDNS multicast → chan   │
        └────────────────────────────┘
```

**Listen goroutine.** Joins the mDNS multicast group and reads raw UDP
packets continuously, decoding each one and handing it off on a
channel — listening never blocks on however busy the rest of the
program is.

**Discovery loop.** Sends the initial "what service types exist"
query, immediately follows up on each new type it learns about, and
cross-references incoming PTR/SRV/A records — since they can arrive in
any order, each one is stored as it's seen and checked against what's
already known, so whichever arrives last is what completes a device's
resolved address.

**Registry + AppState.** The actual source of truth: a TTL-based
registry that infers a device's departure purely from silence past its
own declared trust window, plus a running event log of every
join/leave. Both are safely shared between the background discovery
loop and the UI via a mutex.

**TUI.** A Bubble Tea model reads from the registry and event log
whenever signaled that something changed, and renders the current
state — a live table, an event feed, an optional detail view, and an
optional grouped network map.

## Notable bugs found building this

- **Reverse-DNS PTR records** (`244.1.168.192.in-addr.arpa`) travel as
  the same record type as real service-discovery PTRs, but mean
  something unrelated (an IP-to-hostname lookup). Left unfiltered,
  these showed up as nonsense devices — fixed by recognizing the
  `.in-addr.arpa` / `.ip6.arpa` suffix before treating a PTR as a real
  service.
- **AirPlay-audio (RAOP) names** are prefixed with a device's MAC
  address (`AA8F74BD69AF@Maria's MacBook Air`), which broke device
  grouping — it read as a different device than plain `Maria's
  MacBook Air`. Fixed by stripping everything before `@` when deriving
  a device's group identity.

## Tech stack

| Concern | Choice |
|---|---|
| Language | Go |
| Discovery protocol | mDNS over UDP multicast — hand-decoded DNS wire format, no external DNS library |
| Terminal UI | `charmbracelet/bubbletea`, `bubbles`, `lipgloss` |
| Export format | JSON |

## Installation

### Prerequisites

Just Go — no other system dependencies:

```
brew install go        # macOS
sudo apt install golang # Debian/Ubuntu
```

### Build

```
git clone https://github.com/mdavis1013/pulse.git
cd pulse
go build -o pulse .
```

### Run

Joining a multicast group needs elevated privileges, the same reason
tools like Wireshark need `sudo`:

```
sudo ./pulse
```

## Using it

Pulse boots straight into live discovery. Everything else is driven
by single-key hotkeys shown at the top of the screen:

| Key | Action |
|---|---|
| `↑` `↓` | Move through the device list |
| `enter` | Inspect the selected device (PTR/SRV/A detail + live TTL bars) |
| `esc` | Close the detail view |
| `tab` | Toggle the network map (devices grouped by physical device) |
| `e` | Export the current session to a JSON file |
| `q` | Quit |

## Project status & roadmap

Discovery, liveness tracking, the interactive TUI, device grouping,
and export are complete and verified against a real home network. The
next milestones are TXT record decoding (often carries useful
metadata, like a printer's supported paper sizes), IPv6/AAAA support,
and an automated test suite to replace the current hand-verification
process.
