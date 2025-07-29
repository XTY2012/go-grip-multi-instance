package pkg

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/aarol/reload"
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

	reload := reload.New(directory)
	reload.DebugLog = log.New(io.Discard, "", 0)

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
		urlPath := r.URL.Path

		// Remove leading slash and clean the path
		if urlPath == "/" || urlPath == "" {
			if initialFile != "" {
				urlPath = "/" + initialFile
			} else {
				// Try to find a README.md
				urlPath = "/README.md"
			}
		}

		// Check if the path ends with a directory, look for README.md
		fullPath := filepath.Join(directory, strings.TrimPrefix(urlPath, "/"))
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			// Look for README.md in the directory
			readmePath := filepath.Join(fullPath, "README.md")
			if _, err := os.Stat(readmePath); err == nil {
				// Redirect to the README.md
				relativePath, _ := filepath.Rel(directory, readmePath)
				http.Redirect(w, r, "/"+filepath.ToSlash(relativePath), http.StatusFound)
				return
			}
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
		} else {
			chttp.ServeHTTP(w, r)
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

	handler := reload.Handle(http.DefaultServeMux)
	return http.Serve(listener, handler)
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
