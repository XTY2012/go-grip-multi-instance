package pkg

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	// "github.com/aarol/reload"
	chroma_html "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/chrishrb/go-grip/defaults"
)

type Server struct {
	parser      *Parser
	theme       string
	boundingBox bool
	host        string
	port        int
	browser     bool
}

func NewServer(host string, port int, theme string, boundingBox bool, browser bool, parser *Parser) *Server {
	return &Server{
		host:        host,
		port:        port,
		theme:       theme,
		boundingBox: boundingBox,
		browser:     browser,
		parser:      parser,
	}
}

func (s *Server) Serve(inputPath string) error {
	var directory string
	var initialFile string

	log.Printf("Starting server with inputPath: %s", inputPath)

	// Check if input is a file or directory
	info, err := os.Stat(inputPath)
	if err != nil {
		// If file doesn't exist, check if parent directory exists
		directory = path.Dir(inputPath)
		initialFile = path.Base(inputPath)
		if _, err := os.Stat(directory); err != nil {
			return fmt.Errorf("path not found: %s", inputPath)
		}
	} else if info.IsDir() {
		directory = inputPath
		// Look for README.md as default
		if _, err := os.Stat(filepath.Join(directory, "README.md")); err == nil {
			initialFile = "README.md"
		}
	} else {
		directory = path.Dir(inputPath)
		initialFile = path.Base(inputPath)
	}

	// Convert to absolute path for consistent handling
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	directory = absDir
	log.Printf("Serving directory: %s, initial file: %s", directory, initialFile)

	// Configure reload with more conservative settings
	// Temporarily disable reload for debugging
	// reload := reload.New(directory)
	// reload.DebugLog = log.New(io.Discard, "", 0)

	validThemes := map[string]bool{"light": true, "dark": true, "auto": true}

	if !validThemes[s.theme] {
		log.Println("Warning: Unknown theme ", s.theme, ", defaulting to 'auto'")
		s.theme = "auto"
	}

	dir := http.Dir(directory)
	chttp := http.NewServeMux()
	chttp.Handle("/static/", http.FileServer(http.FS(defaults.StaticFiles)))

	// Regex for markdown
	regex := regexp.MustCompile(`(?i)\.md$`)

	// Serve website with rendered markdown
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Add connection timeout and error recovery
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Recovered from panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		urlPath := r.URL.Path

		// Remove leading slash and clean the path
		if urlPath == "/" || urlPath == "" {
			// For root path, generate TOC for the entire directory
			toc, err := ScanMarkdownFiles(directory)
			if err != nil {
				log.Printf("Error scanning directory: %v", err)
				http.Error(w, "Failed to scan directory", http.StatusInternalServerError)
				return
			}

			// Generate TOC markdown
			tocMarkdown := GenerateTOCMarkdown(toc)
			
			// Parse the TOC markdown to HTML
			htmlContent := s.parser.MdToHTML([]byte(tocMarkdown))
			
			// Serve the TOC page
			err = serveTemplate(w, htmlStruct{
				Content:      string(htmlContent),
				Theme:        s.theme,
				BoundingBox:  s.boundingBox,
				CssCodeLight: getCssCode("github"),
				CssCodeDark:  getCssCode("github-dark"),
			})
			if err != nil {
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
				return
			}
			return
		}

		// Check if the path ends with a directory
		fullPath := filepath.Join(directory, strings.TrimPrefix(urlPath, "/"))
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			// Generate TOC for this subdirectory
			toc, err := ScanMarkdownFiles(fullPath)
			if err != nil {
				log.Printf("Error scanning directory: %v", err)
				http.Error(w, "Failed to scan directory", http.StatusInternalServerError)
				return
			}

			// Generate TOC markdown
			tocMarkdown := GenerateTOCMarkdown(toc)
			
			// Parse the TOC markdown to HTML with proper link transformation
			htmlContent := s.parseMarkdownWithLinks([]byte(tocMarkdown), urlPath)
			
			// Serve the TOC page
			err = serveTemplate(w, htmlStruct{
				Content:      string(htmlContent),
				Theme:        s.theme,
				BoundingBox:  s.boundingBox,
				CssCodeLight: getCssCode("github"),
				CssCodeDark:  getCssCode("github-dark"),
			})
			if err != nil {
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
				return
			}
			return
		}

		// Try to open the file
		f, err := dir.Open(urlPath)
		if err == nil {
			defer f.Close()
		}

		if err == nil && regex.MatchString(urlPath) {
			// Open file and convert to html
			bytes, err := readToString(dir, urlPath)
			if err != nil {
				http.Error(w, "Failed to read file", http.StatusInternalServerError)
				return
			}

			// Parse markdown with link transformation
			htmlContent := s.parseMarkdownWithLinks(bytes, urlPath)

			// Serve
			err = serveTemplate(w, htmlStruct{
				Content:      string(htmlContent),
				Theme:        s.theme,
				BoundingBox:  s.boundingBox,
				CssCodeLight: getCssCode("github"),
				CssCodeDark:  getCssCode("github-dark"),
			})
			if err != nil {
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
				return
			}
		} else if err == nil {
			// Serve static files from the markdown directory
			// Check if it's an image or other static file
			info, err := f.Stat()
			if err != nil {
				http.Error(w, "Failed to stat file", http.StatusInternalServerError)
				return
			}
			
			// Don't serve directories
			if info.IsDir() {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			
			// Serve the file
			http.ServeFile(w, r, filepath.Join(directory, strings.TrimPrefix(urlPath, "/")))
		} else {
			// If file not found and it's a static asset request, serve from embedded files
			if strings.HasPrefix(urlPath, "/static/") {
				chttp.ServeHTTP(w, r)
			} else {
				// For non-static files, return a proper 404
				http.Error(w, "File not found", http.StatusNotFound)
			}
		}
	})

	// Try to find an available port, starting with the requested one
	listener, actualPort, err := s.findAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}

	addr := fmt.Sprintf("http://%s:%d/", s.host, actualPort)
	if initialFile != "" {
		addr = fmt.Sprintf("%s%s", addr, initialFile)
	}

	fmt.Printf("🚀 Starting server: %s\n", addr)

	if s.browser {
		err := Open(addr)
		if err != nil {
			fmt.Println("❌ Error opening browser:", err)
		}
	}

	// Create a server with timeouts to prevent connection exhaustion
	server := &http.Server{
		Handler:      http.DefaultServeMux, // Temporarily disable reload
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	log.Printf("Starting HTTP server on %s", listener.Addr())
	err = server.Serve(listener)
	if err != nil {
		log.Printf("Server error: %v", err)
	}
	return err
}

func readToString(dir http.Dir, filename string) ([]byte, error) {
	f, err := dir.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type htmlStruct struct {
	Content      string
	Theme        string
	BoundingBox  bool
	CssCodeLight string
	CssCodeDark  string
}

func serveTemplate(w http.ResponseWriter, html htmlStruct) error {
	w.Header().Set("Content-Type", "text/html")
	tmpl, err := template.ParseFS(defaults.Templates, "templates/layout.html")
	if err != nil {
		return err
	}
	err = tmpl.Execute(w, html)
	return err
}

func getCssCode(style string) string {
	buf := new(strings.Builder)
	formatter := chroma_html.New(chroma_html.WithClasses(true))
	s := styles.Get(style)
	_ = formatter.WriteCSS(buf, s)
	return buf.String()
}

// parseMarkdownWithLinks processes markdown content and transforms relative links
func (s *Server) parseMarkdownWithLinks(content []byte, currentPath string) []byte {
	// First, preprocess wiki-style links [[text]] -> [text](text.md)
	processedContent := s.preprocessWikiLinks(content)

	// Then parse the markdown to HTML
	htmlContent := s.parser.MdToHTML(processedContent)

	// Transform relative markdown links to work with our server
	// This regex matches markdown links like [text](path.md)
	linkRegex := regexp.MustCompile(`href="([^"]+\.md(?:#[^"]*)?)"`)

	currentDir := path.Dir(currentPath)

	transformed := linkRegex.ReplaceAllFunc(htmlContent, func(match []byte) []byte {
		// Extract the link
		submatch := linkRegex.FindSubmatch(match)
		if len(submatch) < 2 {
			return match
		}

		link := string(submatch[1])

		// Handle absolute paths (starting with /)
		if strings.HasPrefix(link, "/") {
			return match
		}

		// Handle relative paths
		// Resolve the path relative to the current file's directory
		resolvedPath := path.Join(currentDir, link)

		// Make sure the path starts with /
		if !strings.HasPrefix(resolvedPath, "/") {
			resolvedPath = "/" + resolvedPath
		}

		return []byte(fmt.Sprintf(`href="%s"`, resolvedPath))
	})

	return transformed
}

// preprocessWikiLinks converts [[wiki links]] to standard markdown links
func (s *Server) preprocessWikiLinks(content []byte) []byte {
	// Regex to match [[text]] patterns
	wikiLinkRegex := regexp.MustCompile(`\[\[([^\]]+)\]\]`)

	return wikiLinkRegex.ReplaceAllFunc(content, func(match []byte) []byte {
		// Extract the text between [[ and ]]
		submatch := wikiLinkRegex.FindSubmatch(match)
		if len(submatch) < 2 {
			return match
		}

		linkText := string(submatch[1])

		// Convert to filename: lowercase, spaces to hyphens
		filename := strings.ToLower(linkText)
		filename = strings.ReplaceAll(filename, " ", "-")
		// Handle multiple spaces or special characters
		filename = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(filename, "-")
		// Remove leading/trailing hyphens
		filename = strings.Trim(filename, "-")
		// Collapse multiple hyphens
		filename = regexp.MustCompile(`-+`).ReplaceAllString(filename, "-")

		// Wiki links always resolve from root with leading slash
		return []byte(fmt.Sprintf("[%s](/%s.md)", linkText, filename))
	})
}

// findAvailablePort tries to listen on the requested port first,
// if that fails, it tries to find any available port
func (s *Server) findAvailablePort() (net.Listener, int, error) {
	// First, try the requested port
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	listener, err := net.Listen("tcp", addr)
	if err == nil {
		// Successfully bound to the requested port
		return listener, s.port, nil
	}

	// If the requested port is in use, find an available one
	fmt.Printf("⚠️  Port %d is already in use, finding an available port...\n", s.port)

	// Try to bind to port 0, which lets the OS assign an available port
	addr = fmt.Sprintf("%s:0", s.host)
	listener, err = net.Listen("tcp", addr)
	if err != nil {
		return nil, 0, err
	}

	// Get the actual port that was assigned
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		listener.Close()
		return nil, 0, fmt.Errorf("failed to get TCP address")
	}

	return listener, tcpAddr.Port, nil
}
