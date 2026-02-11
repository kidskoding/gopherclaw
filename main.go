package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"gopherclaw/agent"
	"gopherclaw/models"

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

	numWorkers := 3
	var wg sync.WaitGroup
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go agent.GopherWorker(i, &wg, client, tasks, results, quit)
	}

	go func() {
		prompts := []string{
			"Be concise. Explain Go channels in 3 sentences.",
			"Be concise. What is fan-out/fan-in concurrency?",
			"Be concise. Why use goroutines over Python threads?",
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
			fmt.Println("\nShutting down agents...")
			close(quit)
			wg.Wait()
			fmt.Println("GopherClaw stopped.")
			return
		}
	}
}
