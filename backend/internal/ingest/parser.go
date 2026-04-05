package ingest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsawler/tabula"
)

// ParsedDocument represents the result of parsing a document file.
type ParsedDocument struct {
	FileName string
	Pages    []PageContent
}

// PageContent holds the text content for a specific page.
type PageContent struct {
	PageNumber int
	Text       string
}

// SupportedExtensions lists all file types tabula can parse.
var SupportedExtensions = map[string]bool{
	".pdf":  true,
	".docx": true,
	".odt":  true,
	".xlsx": true,
	".pptx": true,
	".html": true,
	".htm":  true,
	".epub": true,
	".txt":  true,
}

// ParseFile extracts text from a document using tabula.
// For .txt files, it reads the content directly.
// Returns a ParsedDocument with per-page content where available.
func ParseFile(filePath string) (*ParsedDocument, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	fileName := filepath.Base(filePath)

	if !SupportedExtensions[ext] {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	// Handle plain text files directly
	if ext == ".txt" {
		return parsePlainText(filePath, fileName)
	}

	// Use tabula for all other formats
	return parseWithTabula(filePath, fileName)
}

// parsePlainText reads a .txt file directly.
func parsePlainText(filePath, fileName string) (*ParsedDocument, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open text file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read text file: %w", err)
	}

	return &ParsedDocument{
		FileName: fileName,
		Pages: []PageContent{
			{PageNumber: 1, Text: string(data)},
		},
	}, nil
}

// parseWithTabula uses tabula to extract text with layout analysis.
func parseWithTabula(filePath, fileName string) (*ParsedDocument, error) {
	ext := tabula.Open(filePath)

	// Get page count (PDFs have multiple pages; others may be single-page)
	pageCount, err := ext.PageCount()
	if err != nil {
		// Fallback: treat as single-page document
		pageCount = 0
	}

	doc := &ParsedDocument{
		FileName: fileName,
		Pages:    make([]PageContent, 0),
	}

	if pageCount > 1 {
		// Extract page by page for multi-page docs (PDFs)
		for i := 1; i <= pageCount; i++ {
			text, _, err := tabula.Open(filePath).
				Pages(i).
				ExcludeHeadersAndFooters().
				JoinParagraphs().
				Text()
			if err != nil {
				// Log warning but continue with other pages
				continue
			}

			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}

			doc.Pages = append(doc.Pages, PageContent{
				PageNumber: i,
				Text:       text,
			})
		}
	} else {
		// Single-page document or non-paginated format
		text, _, err := tabula.Open(filePath).
			ExcludeHeadersAndFooters().
			Text()
		if err != nil {
			return nil, fmt.Errorf("tabula extraction failed for %s: %w", fileName, err)
		}

		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("no text extracted from %s", fileName)
		}

		doc.Pages = append(doc.Pages, PageContent{
			PageNumber: 1,
			Text:       text,
		})
	}

	if len(doc.Pages) == 0 {
		return nil, fmt.Errorf("no text content extracted from %s", fileName)
	}

	return doc, nil
}
