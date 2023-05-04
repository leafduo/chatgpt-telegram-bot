package main

import (
	"fmt"
	"log"
	"net/url"
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
	OPENAIModel                         string  `env:"OPENAI_MODEL" envDefault:"gpt-3.5-turbo"`
	OpenAIBaseURL                       string  `env:"OPENAI_BASE_URL" envDefault:"https://api.openai.com"`
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

func main() {
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}

	baseURL, err := url.JoinPath(cfg.OpenAIBaseURL, "/v1")
	if err != nil {
		panic(err)
	}
	cfg.OpenAIBaseURL = baseURL

	if cfg.OPENAIModel != "gpt-3.5-turbo" && cfg.OPENAIModel != "gpt-4" {
		log.Fatalf("Invalid OPENAI_MODEL: %s", cfg.OPENAIModel)
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
			userID := update.Message.Chat.ID
			msg := update.Message.Text

			wg := conc.NewWaitGroup()
			wg.Go(func() {
				contextTrimmed, err := gpt.SendMessage(userID, msg, answerChan)
				if err != nil {
					log.Print(err)

					_, err = bot.Send(tgbotapi.NewMessage(userID, err.Error()))
					if err != nil {
						log.Print(err)
					}

					return
				}

				if contextTrimmed {
					msg := tgbotapi.NewMessage(userID, "Context trimmed.")
					_, err = bot.Send(msg)
					if err != nil {
						log.Print(err)
					}
				}
			})
			wg.Go(func() {
				lastUpdateTime := time.Now()
				var currentAnswer string
				for answer := range answerChan {
					currentAnswer = answer
					// Update message every 2.5 seconds to avoid hitting Telegram API limits. In the documentation,
					// Although the documentation states that the limit is one message per second, in practice, it is
					// still rate-limited.
					// https://core.telegram.org/bots/faq#my-bot-is-hitting-limits-how-do-i-avoid-this
					if lastUpdateTime.Add(time.Duration(2500) * time.Millisecond).Before(time.Now()) {
						throttledAnswerChan <- currentAnswer
						lastUpdateTime = time.Now()
					}
				}
				throttledAnswerChan <- currentAnswer
				close(throttledAnswerChan)
			})
			wg.Go(func() {
				_, err := bot.Send(tgbotapi.NewChatAction(userID, tgbotapi.ChatTyping))
				if err != nil {
					log.Print(err)
				}

				var messageID int

				for currentAnswer := range throttledAnswerChan {
					if messageID == 0 {
						msg, err := bot.Send(tgbotapi.NewMessage(userID, currentAnswer))
						if err != nil {
							log.Print(err)
						}
						messageID = msg.MessageID
					} else {
						editedMsg := tgbotapi.NewEditMessageText(userID, messageID, currentAnswer)
						_, err := bot.Send(editedMsg)
						if err != nil {
							log.Print(err)
						}
					}
				}
			})

			wg.Wait()
		}
	}
}
