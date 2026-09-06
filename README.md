```
 ____        _
|  _ \ _   _| |___  ___
| |_) | | | | / __|/ _ \
|  __/| |_| | \__ \  __/
|_|    \__,_|_|___/\___|
```

A live mDNS network device scanner with a terminal UI. Discovers real
devices on your WiFi by hand-decoding the DNS wire protocol, built
from scratch in Go.

**Go** · **Multicast** · **TUI**

## About

Every device on your WiFi that supports AirPlay, Chromecast, network
printing, HomeKit, or similar already broadcasts its presence
constantly. Pulse sends the same kind of "please announce yourselves"
query real tools like `dns-sd` use, then decodes every reply itself,
byte by byte, with no DNS library involved. It shows a live,
continuously-updating view of what's actually on the network: which
devices exist, where to reach them, and whether they're still there.

## Features

- **Real mDNS multicast discovery** (`224.0.0.251:5353`). No
  simulated data; every device shown is something actually announcing
  itself on the network at that moment.
- **Hand-rolled DNS wire-format parsing**, including name
  compression, where a record can say "the rest of this name is
  identical to bytes already written earlier in this packet" instead
  of repeating it.
- **Three-record reconstruction**. A device's identity is spread
  across a PTR, an SRV, and an A record, which can arrive
  independently and in any order. Pulse cross-references all three
  into one resolved entry.
- **TTL-based liveness detection**. Nothing in mDNS ever announces
  "I'm leaving." Each device's own declared TTL drives a background
  sweep that infers departure from silence, the same core pattern
  real distributed systems use for failure detection.
- **Device grouping**. Independent service announcements that share
  an underlying device are recognized and grouped as one physical
  device, not several.
- **Live interactive TUI**: a device table, a join/leave event feed,
  a per-record detail view with live-draining TTL bars, and a
  toggleable network map.
- **Session export**. Dump the current device list and event history
  to a readable JSON file.

## How it works

Pulse is organized as four cooperating pieces, all in one Go binary:

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
channel. Listening never blocks on however busy the rest of the
program is.

**Discovery loop.** Sends the initial "what service types exist"
query, immediately follows up on each new type it learns about, and
cross-references incoming PTR/SRV/A records. Since they can arrive in
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
state: a live table, an event feed, an optional detail view, and an
optional grouped network map.

## Tech stack

- **Go**, language
- **mDNS over UDP multicast**, discovery protocol, hand-decoded DNS wire format, no external DNS library
- **`charmbracelet/bubbletea`, `bubbles`, `lipgloss`**, terminal UI
- **JSON**, export format

## Installation

### Prerequisites

Just Go, no other system dependencies:

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

## Controls

Pulse boots straight into live discovery. Everything else is driven
by single-key hotkeys:

| Key | Action |
|---|---|
| `↑` `↓` | Move through the device list |
| `enter` | Inspect the selected device (PTR/SRV/A detail + live TTL bars) |
| `esc` | Close the detail view |
| `tab` | Toggle the network map (devices grouped by physical device) |
| `e` | Export the current session to a JSON file |
| `q` | Quit |

## What's next

Discovery, liveness tracking, the interactive TUI, device grouping,
and export are complete and verified against a real home network.
Still on the list: TXT record decoding (often carries useful metadata,
like a printer's supported paper sizes), IPv6/AAAA support, and an
automated test suite to replace the current hand-verification process.