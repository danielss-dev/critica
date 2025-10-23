# Critica - AI-Powered Git Analysis CLI (Python)

Critica is an AI-powered CLI tool that provides intelligent analysis of your git diffs. This is the Python version focused on AI features.

## Features

- **Comprehensive Analysis**: Get detailed insights about code quality, security, and performance
- **Smart Commit Messages**: Generate conventional commit messages automatically
- **PR Descriptions**: Create detailed pull request descriptions
- All powered by OpenAI's GPT models

## Installation

### Prerequisites

- Python 3.9 or higher
- Git installed on your system
- OpenAI API key

### Install from source

```bash
# Clone the repository
git clone https://github.com/yourusername/critica.git
cd critica

# Install the package
pip install -e .

# Or install dependencies directly
pip install -r requirements.txt
```

## Configuration

### Environment Variables

Set your OpenAI API key:

```bash
export OPENAI_API_KEY="your-api-key-here"
```

Optional configuration:

```bash
export OPENAI_MODEL="gpt-5-nano"  # Default model
export OPENAI_BASE_URL="https://api.openai.com/v1"  # Custom API endpoint
```

### Configuration File

Alternatively, create a configuration file at `~/.config/critica/config.json`:

```json
{
  "ai_enabled": true,
  "openai_api_key": "your-api-key-here",
  "openai_model": "gpt-5-nano",
  "openai_base_url": "https://api.openai.com/v1"
}
```

**Note**: Environment variables take precedence over the configuration file.

## Usage

### Analyze Changes

Perform comprehensive AI analysis of your git diff:

```bash
# Analyze all changes
critica analyze

# Analyze only staged changes
critica analyze --staged

# Analyze specific path
critica analyze /path/to/directory
```

This provides:
- Code quality assessment
- Potential issues and bugs
- Security concerns
- Performance implications
- Improvement suggestions
- Explanations of changes

### Generate Commit Message

Generate a conventional commit message:

```bash
# Generate commit message for staged changes
critica commit

# If no changes are staged, it will automatically stage all modified files
```

The tool will:
1. Analyze your changes
2. Generate a commit message
3. Ask if you want to apply the commit
4. Ask if you want to push to remote

### Generate PR Description

Create a comprehensive pull request description:

```bash
# Interactive mode - select target branch
critica pr

# Specify target branch directly
critica pr main

# Specify path and target branch
critica pr /path/to/repo main
```

This generates:
- Summary of changes
- What was changed and why
- Testing considerations
- Breaking changes (if any)
- Branch context

## Examples

### Analyze Your Changes

```bash
$ critica analyze
🤖 Analyzing changes with AI...

📊 Analysis Results
────────────────────────────────────────────────────

📝 Summary:
Added new feature for user authentication with JWT tokens.
Improved error handling and added comprehensive tests.

🏆 Code Quality:
Good - Code follows best practices with proper error handling

💡 Improvement Suggestions:
  1. Consider adding rate limiting to the auth endpoint
  2. Add input validation for email format
  3. Implement refresh token rotation

🔒 Security Notes:
  1. Ensure JWT secret is stored securely
  2. Consider implementing token expiration
```

### Generate Commit

```bash
$ critica commit
No staged changes found. Staging all modified files...
✅ Files staged successfully!
🤖 Generating commit message...

feat(auth): add JWT-based user authentication

Implement user authentication using JWT tokens with proper
error handling and comprehensive test coverage.

Do you want to apply this commit? [y/N]: y
Applying commit...
✅ Commit applied successfully!

Do you want to push the branch? [y/N]: y
Pushing branch...
✅ Branch pushed successfully!
```

### Generate PR Description

```bash
$ critica pr
Available branches:
  1. main
→ 2. feature/auth (current)
  3. develop

Current branch: feature/auth
Selecting target branch to compare against (where you want to merge into)

Enter target branch number (1-3): 1
Using target branch: main

Comparing feature/auth → main

🤖 Generating PR description...

## Summary

This PR implements JWT-based user authentication for the application.

## Changes

- Added authentication middleware
- Implemented JWT token generation and validation
- Added login and logout endpoints
- Added comprehensive test coverage

## Testing

- Unit tests for auth service
- Integration tests for auth endpoints
- Manual testing with Postman

## Breaking Changes

None

────────────────────────────────────────────────────
```

## Development

### Project Structure

```
critica/
├── critica_py/              # Main package
│   ├── __init__.py
│   ├── cli.py              # CLI interface
│   ├── ai/                 # AI service
│   │   ├── __init__.py
│   │   └── service.py
│   ├── git/                # Git operations
│   │   ├── __init__.py
│   │   ├── diff.py
│   │   ├── branch.py
│   │   └── commit.py
│   └── config/             # Configuration
│       ├── __init__.py
│       └── config.py
├── pyproject.toml          # Project metadata
├── setup.py                # Setup script
└── requirements.txt        # Dependencies
```

### Running Tests

```bash
# Install dev dependencies
pip install -e ".[dev]"

# Run tests (when implemented)
pytest
```

## Migration Notes

This Python version focuses solely on the AI CLI features of Critica:
- `critica analyze` - AI analysis
- `critica commit` - Generate commit messages
- `critica pr` - Generate PR descriptions

The TUI/diff viewer functionality from the Go version has been removed as per the migration requirements.

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
