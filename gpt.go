package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type GPT struct {
	userState map[int64]*UserState
}

type UserState struct {
	TelegramID     int64
	LastActiveTime time.Time
	HistoryMessage []openai.ChatCompletionMessage
}

func NewGPT() *GPT {
	gpt := &GPT{
		userState: make(map[int64]*UserState),
	}

	// TODO: notify expired conversations
	// // check user context expiration every 5 seconds
	// go func() {
	// 	for {
	// 		for userID, user := range users {
	// 			cleared := gpt.clearUserContextIfExpires(userID)
	// 			if cleared {
	// 				lastMessage := user.LatestMessage
	// 				if cfg.NotifyUserOnConversationIdleTimeout {
	// 					msg := tgbotapi.NewEditMessageText(userID, lastMessage.MessageID, lastMessage.Text+"\n\nContext cleared due to inactivity.")
	// 					_, _ = bot.Send(msg)
	// 				}
	// 			}
	// 		}
	// 		time.Sleep(5 * time.Second)
	// 	}
	// }()

	return gpt
}

func (gpt *GPT) SendMessage(userID int64, msg string, answerChan chan<- string) (bool, error) {
	gpt.clearUserContextIfExpires(userID)

	if _, ok := gpt.userState[userID]; !ok {
		gpt.userState[userID] = &UserState{
			TelegramID:     userID,
			LastActiveTime: time.Now(),
			HistoryMessage: []openai.ChatCompletionMessage{},
		}
	}

	user := gpt.userState[userID]

	user.HistoryMessage = append(user.HistoryMessage, openai.ChatCompletionMessage{
		Role:    "user",
		Content: msg,
	})
	user.LastActiveTime = time.Now()

	trimHistory := func() {
		user.HistoryMessage = user.HistoryMessage[1:]
		fmt.Println("History trimmed due to token limit")
	}
	for len(user.HistoryMessage) > 0 {
		tokenCount, err := CountToken(user.HistoryMessage, cfg.OPENAIModel)
		if err != nil {
			fmt.Println("count token error:", err)

			// How should we deal with this error?
			trimHistory()
			continue
		}

		if tokenCount < 3500 {
			break
		}
		trimHistory()
	}

	clientConfig := openai.DefaultConfig(cfg.OpenAIAPIKey)
	clientConfig.BaseURL = cfg.OpenAIBaseURL
	c := openai.NewClientWithConfig(clientConfig)
	ctx := context.Background()

	req := openai.ChatCompletionRequest{
		Model:       cfg.OPENAIModel,
		Temperature: cfg.ModelTemperature,
		TopP:        1,
		N:           1,
		// PresencePenalty:  0.2,
		// FrequencyPenalty: 0.2,
		Messages: user.HistoryMessage,
		Stream:   true,
	}

	stream, err := c.CreateChatCompletionStream(ctx, req)
	if err != nil {
		log.Print(err)
		user.HistoryMessage = user.HistoryMessage[:len(user.HistoryMessage)-1]
		return false, err
	}

	var currentAnswer string

	defer stream.Close()
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			close(answerChan)
			break
		}

		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			close(answerChan)
			break
		}

		currentAnswer += response.Choices[0].Delta.Content
		answerChan <- currentAnswer
	}

	user.HistoryMessage = append(user.HistoryMessage, openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: currentAnswer,
	})

	return false, nil
}

func (gpt *GPT) clearUserContextIfExpires(userID int64) bool {
	user := gpt.userState[userID]
	if user != nil &&
		user.LastActiveTime.Add(time.Duration(cfg.ConversationIdleTimeoutSeconds)*time.Second).Before(time.Now()) {
		gpt.ResetUser(userID)
		return true
	}

	return false
}

func (gpt *GPT) ResetUser(userID int64) {
	delete(gpt.userState, userID)
}
