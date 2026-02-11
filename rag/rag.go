package rag

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/pinecone"
)

type RAG struct {
	store pinecone.Store
}

func New() (*RAG, error) {
	openaiClient, err := openai.New(
		openai.WithEmbeddingModel("text-embedding-ada-002"),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
	)
	if err != nil {
		return nil, fmt.Errorf("openai embedder: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(openaiClient)
	if err != nil {
		return nil, fmt.Errorf("embedder: %w", err)
	}

	store, err := pinecone.New(
		pinecone.WithAPIKey(os.Getenv("PINECONE_API_KEY")),
		pinecone.WithHost(os.Getenv("PINECONE_HOST")),
		pinecone.WithEmbedder(embedder),
		pinecone.WithNameSpace("gopherclaw"),
	)
	if err != nil {
		return nil, fmt.Errorf("pinecone: %w", err)
	}

	return &RAG{store: store}, nil
}

func (r *RAG) Ingest(ctx context.Context, docs []schema.Document) error {
	ids, err := r.store.AddDocuments(ctx, docs)
	if err != nil {
		return fmt.Errorf("ingest: %w", err)
	}
	log.Printf("RAG: ingested %d chunks", len(ids))
	return nil
}

func (r *RAG) Retrieve(ctx context.Context, query string, topK int) string {
	docs, err := r.store.SimilaritySearch(ctx, query, topK)
	if err != nil {
		log.Printf("RAG retrieve error: %v", err)
		return ""
	}

	if len(docs) == 0 {
		return ""
	}

	var parts []string
	for i, doc := range docs {
		parts = append(parts, fmt.Sprintf("[%d] %s", i+1, doc.PageContent))
	}
	return strings.Join(parts, "\n\n")
}
