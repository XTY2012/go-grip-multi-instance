<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="#">
    <img src=".github/docs/logo-1.png" alt="Logo" height="120">
  </a>

  <h3 align="center">go-grip</h3>

  <p align="center">
    Render your markdown files local<br>- with the look of GitHub
  </p>
</div>

## :question: About

**go-grip** is a lightweight, Go-based tool designed to render Markdown files locally, replicating GitHub's style. It offers features like syntax highlighting, dark mode, and support for mermaid diagrams, providing a seamless and visually consistent way to preview Markdown files in your browser.

This project is a reimplementation of the original Python-based [grip](https://github.com/joeyespo/grip), which uses GitHub's web API for rendering. By eliminating the reliance on external APIs, go-grip delivers similar functionality while being fully self-contained, faster, and more secure - perfect for offline use or privacy-conscious users.

## :zap: Features

- :zap: Written in Go :+1:
- 📄 Render markdown to HTML and view it in your browser
- 📁 **Multi-file support** - Serve entire directories of markdown files
- 🔗 **Wiki-style links** - Use `[[Page Name]]` to link to `page-name.md`
- 📱 Dark and light theme
- 🎨 Syntax highlighting for code
- [x] Todo list like the one on GitHub
- Support for github markdown emojis :+1: :bowtie:
- Support for mermaid diagrams
- 🔄 Auto-reload on file changes

```mermaid
graph TD;
      A-->B;
      A-->C;
      B-->D;
      C-->D;
```

> [!TIP]
> Support of blockquotes (note, tip, important, warning and caution) [see here](https://github.com/orgs/community/discussions/16925)


## :rocket: Getting started

### Quick Install

```bash
# Clone and install
git clone https://github.com/chrishrb/go-grip.git
cd go-grip
./install.sh
```

### Alternative Installation Methods

Using Go:
```bash
go install github.com/chrishrb/go-grip@latest
```

> [!TIP]
> You can also use nix flakes to install this plugin.
> More useful information [here](https://nixos.wiki/wiki/Flakes).

## :hammer: Usage

### Basic Usage

```bash
# Serve current directory (looks for README.md)
go-grip

# Serve a specific file
go-grip README.md

# Serve a documentation directory
go-grip docs/
```

The browser will automatically open on http://localhost:6419. You can disable this behaviour with the `-b=false` option.

### Multi-File Documentation

When serving a directory, go-grip supports:
- Automatic README.md detection as the starting page
- Navigation between markdown files in subdirectories
- Relative links between documents
- Auto-reload when files change

### Wiki-Style Links

Use double brackets for easy cross-referencing:

```markdown
[[Getting Started]]     → links to /getting-started.md
[[API Reference]]       → links to /api-reference.md
[[My Complex Title!]]   → links to /my-complex-title.md
```

Wiki links are:
- Case insensitive
- Convert spaces to hyphens
- Always resolve from document root

### Advanced Options

```bash
# Use custom port
go-grip -p 8080 docs/

# Disable browser auto-open
go-grip -b=false

# Set theme (light/dark/auto)
go-grip --theme dark README.md
```

To terminate the current server simply press `CTRL-C`.

## :pencil: Examples

<img src="./.github/docs/example-1.png" alt="examples" width="1000"/>

## :bug: Known TODOs / Bugs

- [ ] Tests and refactoring
- [ ] Make it possible to export the generated html

## :pushpin: Similar tools

This tool is a Go-based reimplementation of the original [grip](https://github.com/joeyespo/grip), offering the same functionality without relying on GitHub's web API.
