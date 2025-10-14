# Critica

A beautiful Git diff viewer for the terminal with split-screen visualization.

## Features

- **Split-screen diff view**: See old and new code side-by-side
- **Unified diff view**: Traditional unified diff format with `-u` flag
- **Interactive mode**: Browse files with fuzzy finder, collapse/expand, and keyboard navigation
- **Shows all changes**: Displays modified, new/untracked, deleted, and renamed files
- **Syntax highlighting**: Automatic language detection and syntax highlighting
- **Color-coded changes**: Red for deletions, green for additions
- **Line numbers**: Clear line numbering on both sides
- **Flexible usage**: View diffs for files, directories, or entire repositories
- **Git integration**: Works with any Git repository

## Installation

### From Source

```bash
go install github.com/danielss-dev/critica@latest
```

### Build Locally

```bash
git clone https://github.com/danielss-dev/critica.git
cd critica
go build -o critica
```

## Usage

### Basic Usage

```bash
# Show diff for entire repository (split-screen view)
critica

# Show diff for a specific file
critica src/main.go

# Show diff for a directory
critica src/

# Show diff in unified format (traditional diff view)
critica --unified

# Interactive mode with fuzzy finder
critica --interactive

# Show staged changes
critica --staged

# Show cached changes (alias for --staged)
critica --cached

# Disable colors
critica --no-color

# Combine flags
critica --interactive --unified
```

### Examples

**View all changes in the repository (split-screen):**
```bash
critica
```

**View changes in unified format:**
```bash
critica -u
```

**Interactive mode - browse files with fuzzy search:**
```bash
critica -i
```

**View changes in a specific file:**
```bash
critica internal/ui/renderer.go
```

**View staged changes before committing:**
```bash
critica --staged
```

### Configuration

Critica loads optional defaults from `~/.config/critica/config.json` (the path provided by `os.UserConfigDir()`). Example:

```json
{
  "interactive": true,
  "unified": false,
  "no_color": false,
  "diff_mode": "all",
  "diff_style": "filled",
  "added_text_color": "#8df0b5",
  "deleted_text_color": "#ff8ba3"
}
```

**Diff style options**

- `default` – transparent background for unchanged lines with colored gutters
- `patch` – classic git patch palette without filled backgrounds
- `filled` – fully colored rows for additions and deletions with a muted neutral background

**Color overrides**

- `added_text_color` – six-digit hex color (with or without `#`) applied to added text and line numbers
- `deleted_text_color` – six-digit hex color applied to deleted text and line numbers

If omitted, the built-in theme colors are used. Invalid hex values are ignored during config normalization.

Command-line flags always override configuration values.

### Interactive Mode

Launch interactive mode with `-i` or `--interactive`:

**Keybindings:**
- `↑/↓` or `j/k` - Navigate files/lines
- `enter` - View selected file's diff
- `space` - Collapse/expand current file
- `tab` - Toggle between split and unified view
- `/` - Search/filter files (fuzzy finder)
- `esc` - Back to file list
- `q` - Quit

**Features:**
- Fuzzy file search with filtering
- Collapsible file diffs
- Toggle between split-screen and unified views on the fly
- Keyboard-driven navigation

## Command-Line Options

| Flag | Short | Description |
|------|-------|-------------|
| `--interactive` | `-i` | Interactive mode with fuzzy finder and collapsible files |
| `--unified` | `-u` | Show unified diff view (non-split) |
| `--staged` | `-s` | Show only staged changes |
| `--cached` | `-c` | Show only cached changes (same as --staged) |
| `--no-color` | | Disable color output |
| `--help` | `-h` | Show help message |

## How It Works

Critica uses:
- **Cobra** for CLI argument parsing
- **Bubble Tea** for interactive TUI mode
- **Bubbles** for UI components (list, fuzzy finder)
- **Lipgloss** for beautiful terminal styling
- **Chroma** for syntax highlighting
- **Git** for retrieving diffs

The tool parses Git diff output (including untracked files) and renders it in either:
- **Split-screen format**: Old code on the left, new code on the right
- **Unified format**: Traditional diff view with +/- prefixes
- **Interactive mode**: Browse files with fuzzy search, collapse/expand diffs, and keyboard navigation

## Project Structure

```
critica/
├── main.go                    # Entry point
├── cmd/
│   └── root.go               # CLI commands and flags
├── internal/
│   ├── git/
│   │   └── diff.go           # Git operations (including untracked files)
│   ├── parser/
│   │   └── parser.go         # Diff parser
│   └── ui/
│       ├── renderer.go       # Split-screen & unified rendering
│       ├── interactive.go    # Interactive TUI mode
│       └── theme.go          # Colors and styling
└── README.md
```

## Requirements

- Go 1.24 or higher
- Git installed and in PATH

## Makefile

This repository includes a `Makefile` with common developer targets:

- `make build` — build the `critica` binary
- `make install` — install the package with `go install`
- `make test` — run `go test ./...`
- `make lint` — run `golangci-lint run` (requires `golangci-lint`)
- `make fmt` — run `go fmt ./...`
- `make vet` — run `go vet ./...`
- `make ci` — run `fmt`, `vet`, `lint`, and `test`

Use `make` or `make all` to build the binary.

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License

## Author

Daniel Schwarz (@danielss-dev)
