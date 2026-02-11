package rag

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tmc/langchaingo/schema"
)

const chunkSize = 500
const chunkOverlap = 50

func LoadTextFile(path string) ([]schema.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	text := string(data)
	chunks := chunkText(text, chunkSize, chunkOverlap)

	var docs []schema.Document
	for i, chunk := range chunks {
		docs = append(docs, schema.Document{
			PageContent: chunk,
			Metadata: map[string]any{
				"source": filepath.Base(path),
				"chunk":  i,
			},
		})
	}

	return docs, nil
}

func chunkText(text string, size, overlap int) []string {
	var chunks []string
	for start := 0; start < len(text); start += size - overlap {
		end := start + size
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[start:end])
		if end == len(text) {
			break
		}
	}
	return chunks
}
