package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/lpernett/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

func main() {
	tasks := make(chan string, 5)
	results := make(chan string, 5)

	godotenv.Load()
	anthropic_api_key := os.Getenv("ANTHROPIC_API_KEY")

	client, err := anthropic.New(
		anthropic.WithModel("claude-sonnet-4-5"),
		anthropic.WithToken(anthropic_api_key),
	)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		for prompt := range tasks {
			res, err := llms.GenerateFromSinglePrompt(
				context.Background(),
				client,
				prompt,
			)
			if err != nil {
				results <- fmt.Errorf("error: %v", err).Error()
				continue
			}
			results <- res
		}
	})

	// test prompt / task
	tasks <- "explain why Go channels are better for AI agents than Python loops."
	close(tasks)

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		fmt.Println("agent output: \n", res)
	}
}