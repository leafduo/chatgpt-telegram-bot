package main

import (
	"fmt"
	"log"
	"os"

	"time"

	"github.com/caarlos0/env/v7"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sourcegraph/conc"
)

var cfg struct {
	TelegramAPIToken                    string  `env:"TELEGRAM_APITOKEN,required"`
	OpenAIAPIKey                        string  `env:"OPENAI_API_KEY,required"`
	ModelTemperature                    float32 `env:"MODEL_TEMPERATURE" envDefault:"1.0"`
	AllowedTelegramID                   []int64 `env:"ALLOWED_TELEGRAM_ID" envSeparator:","`
	ConversationIdleTimeoutSeconds      int     `env:"CONVERSATION_IDLE_TIMEOUT_SECONDS" envDefault:"900"`
	NotifyUserOnConversationIdleTimeout bool    `env:"NOTIFY_USER_ON_CONVERSATION_IDLE_TIMEOUT" envDefault:"false"`
}

type User struct {
	TelegramID     int64
	LastActiveTime time.Time
	HistoryMessage []openai.ChatCompletionMessage
	LatestMessage  tgbotapi.Message
}

var users = make(map[int64]*User)

func main() {
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramAPIToken)
	if err != nil {
		panic(err)
	}

	gpt := NewGPT()

	log.Printf("Authorized on account %s", bot.Self.UserName)

	_, _ = bot.Request(tgbotapi.NewSetMyCommands([]tgbotapi.BotCommand{
		{
			Command:     "help",
			Description: "Get help",
		},
		{
			Command:     "new",
			Description: "Clear context and start a new conversation",
		},
	}...))

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		if len(cfg.AllowedTelegramID) != 0 {
			var userAllowed bool
			for _, allowedID := range cfg.AllowedTelegramID {
				if allowedID == update.Message.Chat.ID {
					userAllowed = true
				}
			}
			if !userAllowed {
				_, err := bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("You are not allowed to use this bot. User ID: %d", update.Message.Chat.ID)))
				if err != nil {
					log.Print(err)
				}
				continue
			}
		}

		if update.Message.IsCommand() { // ignore any non-command Messages
			// Create a new MessageConfig. We don't have text yet,
			// so we leave it empty.
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

			// Extract the command from the Message.
			switch update.Message.Command() {
			case "start":
				msg.Text = "Welcome to ChatGPT bot! Write something to start a conversation. Use /new to clear context and start a new conversation."
			case "help":
				msg.Text = "Write something to start a conversation. Use /new to clear context and start a new conversation."
			case "new":
				gpt.ResetUser(update.Message.From.ID)
				msg.Text = "OK, let's start a new conversation."
			default:
				msg.Text = "I don't know that command"
			}

			if _, err := bot.Send(msg); err != nil {
				log.Print(err)
			}
		} else {
			answerChan := make(chan string)
			throttledAnswerChan := make(chan string)
			var currentAnswer string
			userID := update.Message.Chat.ID
			msg := update.Message.Text

			wg := conc.NewWaitGroup()
			wg.Go(func() {
				err := gpt.SendMessage(userID, msg, answerChan)
				if err != nil {
					log.Print(err)

					_, err = bot.Send(tgbotapi.NewMessage(userID, err.Error()))
					if err != nil {
						log.Print(err)
					}
				}
			})
			wg.Go(func() {
				lastUpdateTime := time.Now()
				for delta := range answerChan {
					currentAnswer += delta

					// Limit message to 1 message per second
					// https://core.telegram.org/bots/faq#my-bot-is-hitting-limits-how-do-i-avoid-this
					if lastUpdateTime.Add(time.Second + time.Millisecond).Before(time.Now()) {
						throttledAnswerChan <- currentAnswer
						lastUpdateTime = time.Now()
					}
				}
				throttledAnswerChan <- currentAnswer
				close(throttledAnswerChan)
			})
			wg.Go(func() {
				msg, err := bot.Send(tgbotapi.NewMessage(userID, "Generating..."))
				if err != nil {
					log.Print(err)
					return
				}

				for currentAnswer := range throttledAnswerChan {
					editedMsg := tgbotapi.NewEditMessageText(userID, msg.MessageID, currentAnswer)
					_, err := bot.Send(editedMsg)
					if err != nil {
						log.Print(err)
					}
				}
			})

			wg.Wait()

			// TODO: count tokens
			// if contextTrimmed {
			// 	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Context trimmed.")
			// 	msg.DisableNotification = true
			// 	err = send(bot, msg)
			// 	if err != nil {
			// 		log.Print(err)
			// 	}
			// }
		}
	}
}
