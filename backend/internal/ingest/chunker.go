package ingest

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// MaxChunkTokens is the maximum token count per chunk (~800 tokens ≈ 3200 chars).
	MaxChunkTokens = 800
	// MaxChunkChars is the character approximation of MaxChunkTokens.
	MaxChunkChars = 3200
	// OverlapChars is the character overlap between consecutive chunks (~200 tokens).
	OverlapChars = 800
)

// Chunk represents a text chunk with metadata for ingestion.
type Chunk struct {
	Text          string
	FileName      string
	PageNumber    int
	ChunkIndex    int
	HeaderContext string
}

// ChunkDocument splits a ParsedDocument into overlapping chunks of ~800 tokens.
// Each chunk is prepended with context (filename + page) for better retrieval.
func ChunkDocument(doc *ParsedDocument) []Chunk {
	var chunks []Chunk
	chunkIdx := 0

	for _, page := range doc.Pages {
		pageChunks := recursiveSplit(page.Text, MaxChunkChars, OverlapChars)

		for _, text := range pageChunks {
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}

			// Create contextual header
			header := fmt.Sprintf("[File: %s | Page: %d]", doc.FileName, page.PageNumber)

			// Prepend context to the chunk content
			contextualText := fmt.Sprintf("%s\n%s", header, text)

			chunks = append(chunks, Chunk{
				Text:          contextualText,
				FileName:      doc.FileName,
				PageNumber:    page.PageNumber,
				ChunkIndex:    chunkIdx,
				HeaderContext: header,
			})
			chunkIdx++
		}
	}

	return chunks
}

// recursiveSplit splits text into chunks using a hierarchy of separators.
// Separators are tried in order: \n\n, \n, ". ", " "
// Falls back to hard character split if no separator produces small enough chunks.
func recursiveSplit(text string, maxChars, overlap int) []string {
	if utf8.RuneCountInString(text) <= maxChars {
		return []string{text}
	}

	separators := []string{"\n\n", "\n", ". ", " "}
	return splitWithSeparators(text, maxChars, overlap, separators)
}

func splitWithSeparators(text string, maxChars, overlap int, separators []string) []string {
	if utf8.RuneCountInString(text) <= maxChars {
		return []string{text}
	}

	if len(separators) == 0 {
		// Hard split by character count
		return hardSplit(text, maxChars, overlap)
	}

	sep := separators[0]
	parts := strings.Split(text, sep)

	var chunks []string
	var current strings.Builder

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		candidateLen := utf8.RuneCountInString(current.String())
		partLen := utf8.RuneCountInString(part)

		if candidateLen > 0 && candidateLen+partLen+len(sep) > maxChars {
			// Current chunk is full — save it
			chunk := strings.TrimSpace(current.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}

			// Start new chunk with overlap from previous
			overlapText := getOverlap(chunk, overlap)
			current.Reset()
			if overlapText != "" {
				current.WriteString(overlapText)
				current.WriteString(sep)
			}
		}

		if current.Len() > 0 {
			current.WriteString(sep)
		}
		current.WriteString(part)
	}

	// Flush remaining
	if current.Len() > 0 {
		remaining := strings.TrimSpace(current.String())
		if remaining != "" {
			// If remaining is still too large, recurse with next separator
			if utf8.RuneCountInString(remaining) > maxChars && len(separators) > 1 {
				subChunks := splitWithSeparators(remaining, maxChars, overlap, separators[1:])
				chunks = append(chunks, subChunks...)
			} else {
				chunks = append(chunks, remaining)
			}
		}
	}

	return chunks
}

// hardSplit splits text by character count as a last resort.
func hardSplit(text string, maxChars, overlap int) []string {
	runes := []rune(text)
	var chunks []string

	for i := 0; i < len(runes); {
		end := i + maxChars
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))

		i = end - overlap
		if i <= 0 && end < len(runes) {
			i = end // Prevent infinite loop
		}
		if end >= len(runes) {
			break
		}
	}

	return chunks
}

// getOverlap returns the last `n` characters of text for overlap.
func getOverlap(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[len(runes)-n:])
}
