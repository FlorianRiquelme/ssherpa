# ssherpa

[![GitHub Release](https://img.shields.io/github/v/release/florianriquelme/ssherpa)](https://github.com/florianriquelme/ssherpa/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

<p align="center"><img src="demo.gif" alt="ssherpa demo" width="800"></p>

**Find and connect to the right SSH server, instantly.**

ssherpa is a terminal UI that helps you manage and connect to SSH servers organized by project. It automatically detects which project you're working in based on your git repository, surfaces the relevant servers, and lets you connect with a single keystroke. Team sharing is powered by credential stores like 1Password — no custom backend needed.

## Features

- **Project-Aware**: Automatically suggests servers based on your current git repository
- **Fuzzy Search**: Find any server instantly by name, hostname, or user
- **SSH Key Selection**: Pick which key to use for each connection
- **1Password Integration**: Manage credentials from 1Password shared vaults
- **Connection History**: Recent connections at your fingertips
- **Config Management**: Add, edit, and delete SSH connections from the TUI
- **Zero Dependencies**: Single binary, instant startup

## Installation

### Homebrew

```sh
brew tap ssherpa/tap
brew install ssherpa
```

### Quick Install (macOS/Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/florianriquelme/ssherpa/main/scripts/install.sh | sh
```

### GitHub Releases

Download pre-built binaries directly from the [releases page](https://github.com/florianriquelme/ssherpa/releases). Verify the SHA256 checksum after downloading.

## Quick Start

```sh
# Launch ssherpa
ssherpa

# Re-run setup wizard
ssherpa --setup

# Show version info
ssherpa --version
```

## Usage

| Key | Action |
|-----|--------|
| `j/k` or arrow keys | Navigate servers |
| `/` | Search |
| `Enter` | Connect via SSH |
| `d` | Show server details |
| `a` | Add new server |
| `e` | Edit server |
| `x` | Delete server |
| `p` | Assign project |
| `K` | Change SSH key |
| `q` | Quit |

## Configuration

ssherpa stores its configuration in `~/.config/ssherpa/config.toml`.

Backend options:
- **SSH config only**: Read from `~/.ssh/config` (read-only)
- **1Password**: Sync servers with 1Password shared vaults
- **Both**: Combine SSH config with 1Password backend

Additional settings:
- `ReturnToTUI`: Return to the TUI after SSH session ends (default: false)

Run `ssherpa --setup` to reconfigure backends at any time.

## License

MIT License - see [LICENSE](LICENSE) for details.
