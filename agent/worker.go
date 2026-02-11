package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gopherclaw/models"
	"gopherclaw/tools"

	"github.com/tmc/langchaingo/llms"
)

const maxToolRounds = 15

func GopherWorker(
	id int,
	wg *sync.WaitGroup,
	client llms.Model,
	tasks <-chan models.Task,
	results chan<- models.Result,
	quit <-chan struct{},
) {
	defer wg.Done()
	availableTools := tools.Definitions()

	for {
		select {
		case task, ok := <-tasks:
			if !ok {
				return
			}

			content, err := runAgentLoop(client, task, availableTools)
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

func runAgentLoop(client llms.Model, task models.Task, availableTools []llms.Tool) (string, error) {
	prompt := task.Prompt
	if task.Context != "" {
		prompt = fmt.Sprintf("Context: %s\n\n%s", task.Context, task.Prompt)
	}

	messages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(prompt)},
		},
	}

	for range maxToolRounds {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		resp, err := client.GenerateContent(ctx, messages,
			llms.WithTools(availableTools),
			llms.WithMaxTokens(1024),
		)
		cancel()

		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no response from model")
		}

		var textContent string
		var allToolCalls []llms.ToolCall
		for _, choice := range resp.Choices {
			if choice.Content != "" {
				textContent = choice.Content
			}
			allToolCalls = append(allToolCalls, choice.ToolCalls...)
		}

		if len(allToolCalls) == 0 {
			return textContent, nil
		}
		
		tc := allToolCalls[0]

		messages = append(messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{tc},
		})

		toolName := tc.FunctionCall.Name
		toolArgs := tc.FunctionCall.Arguments

		fmt.Printf("  [tool] %s(%s)\n", toolName, toolArgs)
		result := tools.Execute(toolName, toolArgs)

		messages = append(messages, llms.MessageContent{
			Role: llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{
				llms.ToolCallResponse{
					ToolCallID: tc.ID,
					Name:       toolName,
					Content:    result,
				},
			},
		})
	}

	return "", fmt.Errorf("agent hit max tool rounds (%d)", maxToolRounds)
}
