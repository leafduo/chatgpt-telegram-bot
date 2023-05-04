package main

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
	"github.com/sashabaranov/go-openai"
)

func CountToken(messages []openai.ChatCompletionMessage, model string) (int, error) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return 0, err
	}

	var tokensPerMessage int
	var tokensPerName int
	if model == "gpt-3.5-turbo-0301" || model == "gpt-3.5-turbo" {
		tokensPerMessage = 4
		tokensPerName = -1
	} else if model == "gpt-4-0314" || model == "gpt-4" {
		tokensPerMessage = 3
		tokensPerName = 1
	} else {
		fmt.Println("Warning: model not found. Using cl100k_base encoding.")
		tokensPerMessage = 3
		tokensPerName = 1
	}

	var tokenCount int

	for _, message := range messages {
		tokenCount += tokensPerMessage
		tokenCount += len(tkm.Encode(message.Content, nil, nil))
		tokenCount += len(tkm.Encode(message.Role, nil, nil))
		if message.Name != "" {
			tokenCount += tokensPerName
		}
	}
	tokenCount += 3
	return tokenCount, nil
}
