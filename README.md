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
- **🤖 AI-Powered Analysis**: Get intelligent insights about your code changes
  - **Code quality analysis**: Comprehensive assessment of code quality
  - **Improvement suggestions**: Specific, actionable recommendations
  - **Bug detection**: Identify potential issues and security concerns
  - **Commit message generation**: AI-generated conventional commit messages
  - **PR description generation**: Detailed pull request descriptions
  - **Change explanations**: Clear explanations of what changed and why

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

# Enable AI analysis
critica --ai

# Combine flags
critica --interactive --unified --ai
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

**AI-Powered Analysis:**
```bash
# Comprehensive AI analysis
critica ai analyze

# Generate commit message
critica ai commit

# Generate PR description
critica ai pr

# Get improvement suggestions
critica ai improve

# Explain code changes
critica ai explain
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

### AI Configuration

Critica supports AI-powered analysis using OpenAI's API. To enable AI features:

1. **Set your OpenAI API key:**
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

2. **Optional configuration in `~/.config/critica/config.json`:**
   ```json
   {
     "ai_enabled": true,
     "openai_model": "gpt-5-nano",
     "openai_base_url": "https://api.openai.com/v1"
   }
   ```

3. **Use AI features:**
   ```bash
   # Enable AI in interactive mode
   critica --ai
   
   # Or use AI commands directly
   critica ai analyze
   ```

**Available AI Models:**
- `gpt-5-nano` (default) - Fast and cost-effective
- `gpt-4o` - More capable but slower
- `gpt-3.5-turbo` - Alternative option

**AI Features:**
- **Analysis**: Comprehensive code quality, security, and performance analysis
- **Improvements**: Specific suggestions for code enhancement
- **Commit Messages**: Conventional commit message generation
- **PR Descriptions**: Detailed pull request descriptions
- **Explanations**: Clear explanations of code changes

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

**AI Keybindings (when `--ai` flag is used):**
- `1` - AI Analysis - Comprehensive code analysis
- `2` - AI Commit - Generate commit message
- `3` - AI PR - Generate PR description
- `4` - AI Improve - Get improvement suggestions
- `5` - AI Explain - Explain code changes
- `r` - Retry AI operation (in AI views)
- `esc` - Back to file list (in AI views)

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
| `--ai` | | Enable AI analysis and suggestions |
| `--help` | `-h` | Show help message |

### AI Commands

| Command | Description |
|---------|-------------|
| `critica ai analyze [path]` | Perform comprehensive AI analysis of git diff |
| `critica ai commit [path]` | Generate conventional commit message |
| `critica ai pr [path]` | Generate PR description |
| `critica ai improve [path]` | Get code improvement suggestions |
| `critica ai explain [path]` | Explain code changes |

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
