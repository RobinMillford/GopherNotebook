package ingest

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var urlHTTPClient = &http.Client{Timeout: 30 * time.Second}

// FetchURL downloads a URL, writes the body to a temp .html file, parses it
// with tabula (which already supports HTML), and returns a ParsedDocument.
func FetchURL(rawURL string) (*ParsedDocument, error) {
	resp, err := urlHTTPClient.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
	}

	tmp, err := os.CreateTemp("", "gn-url-*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, io.LimitReader(resp.Body, 10<<20)); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("failed to write response: %w", err)
	}
	tmp.Close()

	doc, err := ParseFile(tmp.Name())
	if err != nil {
		return nil, err
	}
	doc.FileName = URLToFileName(rawURL)
	return doc, nil
}

// URLToFileName derives a safe display filename from a URL.
func URLToFileName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "url_source.html"
	}
	host := u.Hostname()
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return host + ".html"
	}
	parts := strings.Split(path, "/")
	last := parts[len(parts)-1]
	if last == "" || !strings.Contains(last, ".") {
		slug := strings.ReplaceAll(path, "/", "_")
		last = host + "_" + slug
	}
	if len(last) > 60 {
		last = last[:60]
	}
	if !strings.HasSuffix(last, ".html") {
		last += ".html"
	}
	return last
}
