# WakeUp

> Terminal-based Wake-on-LAN utility — fast, keyboard-driven, single binary.

WakeUp is a [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI application for managing devices and sending Wake-on-LAN magic packets. Devices are stored locally, navigation follows Vim conventions, and the theme adapts to both light and dark terminal backgrounds automatically.

[中文](README_CN.md)

## Features

- **Wake-on-LAN** — Select a device and press `Enter` to send a magic packet. Result is displayed asynchronously without blocking the UI.
- **Device Management** — Add, edit, and delete devices. Each entry stores a name, MAC address, port, and broadcast address.
- **Vim-style Navigation** — `j`/`k` to browse, `dd` to delete, `a` to add, `e` to edit. `Esc` always steps back to the previous context.
- **Adaptive Theming** — Uses `lipgloss.AdaptiveColor` with 16 ANSI colors. No hardcoded hex values — follows your terminal's light/dark mode automatically.
- **Single Binary** — One statically-compiled file. No runtime dependencies, no daemon, no configuration files.

## Installation

### Download from Releases

The simplest way — grab a prebuilt binary, no Go toolchain needed.

1. Go to the [Releases page](https://github.com/rainLyn/WakeUp/releases)
2. Download the binary that matches your platform:

| Your Machine | Download |
|-------------|----------|
| macOS Intel | `wakeup-darwin-amd64-v*` |
| macOS Apple Silicon | `wakeup-darwin-arm-v*` |
| Linux x86_64 | `wakeup-linux-amd64-v*` |
| Linux ARM | `wakeup-linux-arm-v*` |
| Windows x86_64 | `wakeup-windows-amd64-v*.exe` |

**macOS / Linux:**

```bash
# Make it executable
chmod +x wakeup-*
# Optional — rename and drop it anywhere on your $PATH
sudo mv wakeup-*-v* /usr/local/bin/wakeup
```

Or just run it in place:

```bash
./wakeup-darwin-arm-v0.1.0
```

**Windows:** Rename to `wakeup.exe` and run it from cmd / PowerShell, or place it in a directory listed in `%PATH%`.

### Build from Source

#### Prerequisites

[Go](https://go.dev/dl/) 1.26 or later.

```bash
git clone https://github.com/rainLyn/WakeUp.git
cd WakeUp
make
```

The resulting binary is `wakeup`. To install system-wide:

```bash
make install
```

#### Makefile Targets

| Target | Description |
|--------|-------------|
| `make` | Build the `wakeup` binary |
| `make run` | Build and launch |
| `make clean` | Remove build artifacts |
| `make install` | Install to `/usr/local/bin` |

## Usage

```bash
./wakeup
```

### Key Bindings

| Context | Key | Action |
|---------|-----|--------|
| Device List | `j` / `k` / `↑` / `↓` | Move selection |
| Device List | `Enter` | Wake selected device |
| Device List | `a` | Add a new device |
| Device List | `e` | Edit selected device |
| Device List | `dd` | Delete selected device |
| Global | `Esc` | Close modal / return to previous page |
| Global | `Ctrl+C` | Quit |

## Data Storage

Device data is persisted to `~/.wakeup/devices.json`:

```json
[
  {
    "name": "NAS",
    "mac": "AA:BB:CC:DD:EE:FF",
    "port": 9,
    "address": "255.255.255.255"
  }
]
```

- Created automatically on first run. Corrupted data files are reset to an empty list.
- **Asynchronous writes** — file I/O never blocks UI rendering.
- **In-memory cache** — reads are instant; writes flush to disk in the background.

## MAC Address Format

WakeUp normalizes all MAC addresses to `XX:XX:XX:XX:XX:XX`. The following formats are accepted as input:

| Input | Normalized |
|-------|-----------|
| `AA:BB:CC:DD:EE:FF` | `AA:BB:CC:DD:EE:FF` |
| `aa-bb-cc-dd-ee-ff` | `AA:BB:CC:DD:EE:FF` |
| `AABB.CCDD.EEFF` | `AA:BB:CC:DD:EE:FF` |
| `AABBCCDDEEFF` | `AA:BB:CC:DD:EE:FF` |

## Project Structure

```
WakeUp/
├── main.go          # Entry point, top-level Bubble Tea model
├── Makefile         # Build automation
├── store/
│   └── store.go     # Device CRUD, JSON persistence, field validation
├── wol/
│   └── wol.go       # Magic packet construction and UDP transport
└── ui/
    ├── ui.go        # Shared styles, modals, help bar, layout helpers
    ├── list.go      # Device list (main view) with inline wake/delete modals
    ├── form.go      # Add/edit form (four fields)
    └── menu.go      # Classic menu page (optional entry point)
```

### Architecture

- **Page-autonomous routing** — each page's `Update()` returns the next page model directly. No central router, no `switch pageType`.
- **Vim modal system** — each page manages its own modal state (Normal, Confirm, WakeConfirm, Result). `Esc` is the universal back action.
- **Async via `tea.Cmd`** — WOL packet transmission and file writes run in goroutines. Results are delivered through typed messages.
- **Adaptive theming** — all styles use `lipgloss.AdaptiveColor` with terminal ANSI colors, respecting the terminal's light/dark mode.
