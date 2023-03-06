package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
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

func (gpt *GPT) SendMessage(userID int64, msg string, answerChan chan<- string) error {
	gpt.clearUserContextIfExpires(userID)

	if _, ok := users[userID]; !ok {
		users[userID] = &User{
			TelegramID:     userID,
			LastActiveTime: time.Now(),
			HistoryMessage: []openai.ChatCompletionMessage{},
		}
	}

	users[userID].HistoryMessage = append(users[userID].HistoryMessage, openai.ChatCompletionMessage{
		Role:    "user",
		Content: msg,
	})
	users[userID].LastActiveTime = time.Now()

	c := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Temperature: cfg.ModelTemperature,
		TopP:        1,
		N:           1,
		// PresencePenalty:  0.2,
		// FrequencyPenalty: 0.2,
		Messages: users[userID].HistoryMessage,
		Stream:   true,
	}

	stream, err := c.CreateChatCompletionStream(ctx, req)
	if err != nil {
		log.Print(err)
		users[userID].HistoryMessage = users[userID].HistoryMessage[:len(users[userID].HistoryMessage)-1]
		return err
	}

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

		fmt.Printf("%+v\n", response)
		answerChan <- response.Choices[0].Delta.Content
	}

	return nil

	// answer := resp.Choices[0].Message

	// users[userID].HistoryMessage = append(users[userID].HistoryMessage, answer)

	// var contextTrimmed bool
	// if resp.Usage.TotalTokens > 3500 {
	// 	users[userID].HistoryMessage = users[userID].HistoryMessage[1:]
	// 	contextTrimmed = true
	// }

	// return answer.Content, contextTrimmed, nil
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
