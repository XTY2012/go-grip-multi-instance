# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

go-grip is a lightweight Go tool for rendering Markdown files locally with GitHub-style formatting. It's a self-contained reimplementation of the Python grip tool without external API dependencies.

## Key Commands

### Development
- `make run [args]` - Run the application with arguments (e.g., `make run README.md`)
- `make build` - Build the binary to `bin/go-grip`
- `make test` - Run all tests
- `make format` - Format code using `go fmt`
- `make lint` - Run golangci-lint
- `make all` - Format, lint, and build

### Cross-compilation
- `make compile` - Build for multiple platforms (darwin/linux/windows, amd64/arm64)

### Utilities
- `make emojiscraper` - Update emoji mappings from GitHub

## Architecture

### Core Components
- **cmd/**: CLI commands using Cobra framework
  - `root.go`: Main command with flags for port, theme, auto-open, etc.
  - `emojiscraper.go`: Debug command for updating emoji data
  
- **pkg/**: Main application logic
  - `server.go`: HTTP server with file watching and auto-reload
  - `parser.go`: Markdown-to-HTML conversion with custom rendering
  - `open.go`: Cross-platform browser opening
  
- **defaults/**: Embedded static assets using Go's embed package
  - Templates, CSS, JavaScript, images, and emojis

### Parser Extensions
The markdown parser (`pkg/parser.go`) implements custom rendering for:
- Syntax highlighting via Chroma
- Mermaid diagram support
- GitHub-style alerts (note, tip, important, warning, caution)
- Emoji replacement using generated mappings
- Task list checkboxes
- Proper heading anchors with GitHub-compatible IDs

### Server Features
- Serves on port 6419 by default (configurable via `-p`)
- Auto-opens browser (disable with `--no-browser`)
- File watching with auto-reload
- Theme support: light, dark, auto (via `-t`)
- Serves both markdown files and static assets

## Testing
Run individual tests:
```bash
go test ./pkg -run TestFunctionName
go test ./cmd -run TestFunctionName
```

## Important Patterns
- All static assets are embedded in the binary using `//go:embed`
- Custom render hooks extend gomarkdown's HTML renderer
- The server uses Go templates for HTML generation
- Cross-platform support requires testing on Windows, macOS, and Linux