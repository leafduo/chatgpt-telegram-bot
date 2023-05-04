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
	HistoryMessage []Message
}

type Message struct {
	Role       string
	Content    string
	TokenCount int
}

func convertMessageToChatCompletionMessage(msg []Message) []openai.ChatCompletionMessage {
	var result []openai.ChatCompletionMessage
	for _, m := range msg {
		result = append(result, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
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
			HistoryMessage: []Message{},
		}
	}

	user := gpt.userState[userID]

	userTokenCount, err := CountToken(msg)
	if err != nil {
		log.Print(err)
	}
	user.HistoryMessage = append(user.HistoryMessage, Message{
		Role:       "user",
		Content:    msg,
		TokenCount: userTokenCount,
	})
	user.LastActiveTime = time.Now()

	c := openai.NewClient(cfg.OpenAIAPIKey)
	ctx := context.Background()

	log.Print(user.HistoryMessage)

	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Temperature: cfg.ModelTemperature,
		TopP:        1,
		N:           1,
		// PresencePenalty:  0.2,
		// FrequencyPenalty: 0.2,
		Messages: convertMessageToChatCompletionMessage(user.HistoryMessage),
		Stream:   true,
	}

	stream, err := c.CreateChatCompletionStream(ctx, req)
	if err != nil {
		log.Print(err)
		user.HistoryMessage = user.HistoryMessage[:len(user.HistoryMessage)-1]
		return false, err
	}

	var currentAnswer string
	var assistantTokenCount int

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

		// It seems that OpenAI sends one token per event, so we can count the tokens by the number of events we
		// receive.
		assistantTokenCount++
		currentAnswer += response.Choices[0].Delta.Content
		answerChan <- currentAnswer
	}

	user.HistoryMessage = append(user.HistoryMessage, Message{
		Role:       "assistant",
		Content:    currentAnswer,
		TokenCount: assistantTokenCount,
	})

	var totalTokenCount int
	for i := len(user.HistoryMessage) - 1; i >= 0; i-- {
		totalTokenCount += user.HistoryMessage[i].TokenCount
		if totalTokenCount > 3500 {
			user.HistoryMessage = user.HistoryMessage[i+1:]
			return true, nil
		}
	}

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
