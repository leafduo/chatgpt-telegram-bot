# ChatGPT Telegram bot

Run your own ChatGPT Telegram bot!

## Setup

1. Get your OpenAI API key

   You can create an account on the OpenAI website and [generate your API key](https://platform.openai.com/account/api-keys).

2. Get your telegram bot token

   Create a bot from Telegram [@BotFather](https://t.me/BotFather) and obtain an access token.

3. Install using `go install`

   If you have a Go environment, you can install it with the following command:

```bash
go install github.com/leafduo/chatgpt-telegram-bot@latest
```

4. Install using binary

   You can get prebuilt binaries from [GitHub Releases](https://github.com/leafduo/chatgpt-telegram-bot/releases) and put it in `$PATH`

5. Install using Docker-compose

   Check out [docker-compose.yml](docker-compose.yml) for sample config

6. Set the environment variables and run

```bash
export OPENAI_API_KEY=<your_openai_api_key>
export TELEGRAM_APITOKEN=<your_telegram_bot_token>
# Optional, default is empty. Only allow these users to use the bot. Empty means allow all users.
export ALLOWED_TELEGRAM_ID=<your_telegram_id>,<your_friend_telegram_id>
# Optional, default is 1.0. Higher temperature means more random responses.
# See https://platform.openai.com/docs/api-reference/chat/create#chat/create-temperature
export MODEL_TEMPERATURE=1.0
# Optional, default is 900. Max idle duration for a certain conversation.
# After this duration, a new conversation will be started.
export CONVERSATION_IDLE_TIMEOUT_SECONDS=900
# Optional, defaults to gpt-3.5-turbo. Specify which model to use.
# Currently, only `gpt-3.5-turbo` and `gpt-4` are supported.
export OPENAIModel=gpt-3.5-turbo
# Optional, defaults to https://api.openai.com.
# You can use this to set a custom OpenAI API endpoint to use third party relay services like https://api2d.com/.
export OpenAIBaseURL=https://api.openai.com

chatgpt-telegram-bot
```
