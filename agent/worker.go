package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gopherclaw/models"

	"github.com/tmc/langchaingo/llms"
)

func GopherWorker(
	id int,
	wg *sync.WaitGroup,
	client llms.Model,
	tasks <-chan models.Task,
	results chan<- models.Result,
	quit <-chan struct{},
) {
	defer wg.Done()
	for {
		select {
		case task, ok := <-tasks:
			if !ok {
				return
			}

			prompt := task.Prompt
			if task.Context != "" {
				prompt = fmt.Sprintf("context: %s\n\n%s", task.Context, task.Prompt)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			content, err := llms.GenerateFromSinglePrompt(ctx, client, prompt, llms.WithMaxTokens(1024))
			cancel()

			results <- models.Result{
				WorkerID: id,
				TaskID:   task.ID,
				Content:  content,
				Error:    err,
			}

		case <-quit:
			return
		}
	}
}
