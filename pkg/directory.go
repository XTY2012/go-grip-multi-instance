package pkg

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MarkdownFile represents a markdown file found during directory scanning
type MarkdownFile struct {
	Path         string // Relative path from the base directory
	Title        string // File name without extension
	FullPath     string // Absolute path to the file
	IsIndex      bool   // True if this is a README.md or index.md
	DirectoryLevel int  // How deep in the directory structure
}

// DirectoryTOC represents the table of contents for a directory
type DirectoryTOC struct {
	BasePath  string
	Files     []MarkdownFile
	HasReadme bool
	Readme    *MarkdownFile
}

// ScanMarkdownFiles recursively scans a directory for markdown files
func ScanMarkdownFiles(basePath string) (*DirectoryTOC, error) {
	toc := &DirectoryTOC{
		BasePath: basePath,
		Files:    []MarkdownFile{},
	}

	// Walk the directory tree
	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and .git
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a markdown file
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		// Get relative path from base
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		// Calculate directory level
		level := strings.Count(relPath, string(os.PathSeparator))

		// Get title (filename without extension)
		title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		
		// Check if it's an index file
		isIndex := strings.EqualFold(d.Name(), "README.md") || strings.EqualFold(d.Name(), "index.md")

		mdFile := MarkdownFile{
			Path:         relPath,
			Title:        title,
			FullPath:     path,
			IsIndex:      isIndex,
			DirectoryLevel: level,
		}

		// Special handling for README.md in the root directory
		if relPath == "README.md" || relPath == "index.md" {
			toc.HasReadme = true
			toc.Readme = &mdFile
		}

		toc.Files = append(toc.Files, mdFile)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	// Sort files: directories first, then alphabetically
	sort.Slice(toc.Files, func(i, j int) bool {
		// Get directory paths
		dirI := filepath.Dir(toc.Files[i].Path)
		dirJ := filepath.Dir(toc.Files[j].Path)

		// If same directory, sort by name (README/index first)
		if dirI == dirJ {
			if toc.Files[i].IsIndex != toc.Files[j].IsIndex {
				return toc.Files[i].IsIndex
			}
			return strings.ToLower(toc.Files[i].Title) < strings.ToLower(toc.Files[j].Title)
		}

		// Otherwise sort by directory path
		return dirI < dirJ
	})

	return toc, nil
}

// GenerateTOCMarkdown generates markdown content for the directory TOC
func GenerateTOCMarkdown(toc *DirectoryTOC) string {
	var sb strings.Builder

	// Title
	sb.WriteString("# 📁 Directory Contents\n\n")
	
	// Show base path
	sb.WriteString(fmt.Sprintf("**Base Path:** `%s`\n\n", toc.BasePath))
	
	// Statistics
	sb.WriteString(fmt.Sprintf("**Total Markdown Files:** %d\n\n", len(toc.Files)))
	
	// Separator
	sb.WriteString("---\n\n")

	// If there's a README in the root, show it prominently
	if toc.HasReadme && toc.Readme != nil {
		sb.WriteString("## 📄 Main Documentation\n\n")
		sb.WriteString(fmt.Sprintf("- [**%s**](%s) (Project README)\n\n", toc.Readme.Title, "/"+filepath.ToSlash(toc.Readme.Path)))
	}

	// Group files by directory
	filesByDir := make(map[string][]MarkdownFile)
	for _, file := range toc.Files {
		dir := filepath.Dir(file.Path)
		filesByDir[dir] = append(filesByDir[dir], file)
	}

	// Get sorted directory list
	var dirs []string
	for dir := range filesByDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	// Generate TOC grouped by directory
	sb.WriteString("## 📚 All Markdown Files\n\n")
	
	for _, dir := range dirs {
		files := filesByDir[dir]
		
		// Directory header
		if dir == "." {
			sb.WriteString("### 📂 Root Directory\n\n")
		} else {
			// Clean up the directory path for display
			displayDir := filepath.ToSlash(dir)
			sb.WriteString(fmt.Sprintf("### 📂 %s\n\n", displayDir))
		}

		// List files in this directory
		for _, file := range files {
			// Indent based on file name
			indent := ""
			
			// Create the link - use forward slashes for web paths
			webPath := "/" + filepath.ToSlash(file.Path)
			
			// Add emoji for different file types
			emoji := "📄"
			if file.IsIndex {
				emoji = "📖"
			}
			
			// Format the title
			displayTitle := file.Title
			if file.IsIndex {
				displayTitle = fmt.Sprintf("**%s**", displayTitle)
			}
			
			sb.WriteString(fmt.Sprintf("%s- %s [%s](%s)\n", indent, emoji, displayTitle, webPath))
		}
		sb.WriteString("\n")
	}

	// Add a footer with navigation help
	sb.WriteString("---\n\n")
	sb.WriteString("## 🔍 Navigation Tips\n\n")
	sb.WriteString("- Click any file name to view its rendered content\n")
	sb.WriteString("- Use your browser's back button to return to this index\n")
	sb.WriteString("- Files are organized by directory structure\n")
	sb.WriteString("- **Bold** entries are README or index files\n")

	return sb.String()
}