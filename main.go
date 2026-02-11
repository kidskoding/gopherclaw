package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"gopherclaw/agent"
	"gopherclaw/mcp"
	"gopherclaw/models"
	"gopherclaw/rag"
	"gopherclaw/tools"

	"github.com/lpernett/godotenv"
	"github.com/tmc/langchaingo/llms/anthropic"
)

func main() {
	tasks := make(chan models.Task, 10)
	results := make(chan models.Result, 10)
	quit := make(chan struct{})

	godotenv.Load()
	anthropicAPIKey := os.Getenv("ANTHROPIC_API_KEY")

	client, err := anthropic.New(
		anthropic.WithModel("claude-sonnet-4-5"),
		anthropic.WithToken(anthropicAPIKey),
	)
	if err != nil {
		log.Fatal(err)
	}

	if cfg, err := mcp.LoadConfig("mcp_servers.json"); err == nil && len(cfg.Servers) > 0 {
		mcpMgr, err := mcp.NewManager(context.Background(), cfg.Servers)
		if err != nil {
			log.Printf("MCP init warning: %v", err)
		} else {
			tools.SetMCPManager(mcpMgr)
			defer mcpMgr.Close()
		}
	}

	if os.Getenv("PINECONE_API_KEY") != "" && os.Getenv("PINECONE_HOST") != "" {
		r, err := rag.New()
		if err != nil {
			log.Printf("RAG init warning: %v", err)
		} else {
			tools.SetRAG(r)
			log.Println("RAG: connected to Pinecone")

			if ingestPath := os.Getenv("INGEST_PATH"); ingestPath != "" {
				docs, err := rag.LoadTextFile(ingestPath)
				if err != nil {
					log.Fatalf("RAG ingest error: %v", err)
				}
				if err := r.Ingest(context.Background(), docs); err != nil {
					log.Fatalf("RAG ingest error: %v", err)
				}
			}
		}
	}

	numWorkers := 1
	var wg sync.WaitGroup
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go agent.GopherWorker(i, &wg, client, tasks, results, quit)
	}

	go func() {
		prompts := []string{
			"Who created GopherClaw and what tools does it support? Search the knowledge base first, then verify by reading the source code.",
		}
		
		for i, p := range prompts {
			tasks <- models.Task{ID: i, Prompt: p}
		}
		
		close(tasks)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case res, ok := <-results:
			if !ok {
				fmt.Println("\nAll tasks complete.")
				return
			}
			if res.Error != nil {
				fmt.Printf("\n[Worker %d | Task %d] ERROR: %v\n", res.WorkerID, res.TaskID, res.Error)
			} else {
				fmt.Printf("\n[Worker %d | Task %d]\n%s\n", res.WorkerID, res.TaskID, res.Content)
			}
		case <-sig:
			close(quit)
			wg.Wait()
			return
		}
	}
}
