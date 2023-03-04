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

   You can get prebuilt binaries from [Github Releases](https://github.com/leafduo/chatgpt-telegram-bot/releases) and put it in `$PATH`

5. Install using Docker-compose

   Check out [docker-compose.yml](docker-compose.yml) for sample config

6. Set the environment variables and run

```bash
export OPENAI_API_KEY=<your_openai_api_key>
export TELEGRAM_APITOKEN=<your_telegram_bot_token>
export ALLOWED_TELEGRAM_ID=<your_telegram_id>,<your_friend_telegram_id>    # optional, default is empty. Only allow these users to use the bot. Empty means allow all users.
export MODEL_TEMPERATURE=1.0  # optional, default is 1.0. Higher temperature means more random responses. See https://platform.openai.com/docs/api-reference/chat/create#chat/create-temperature

chatgpt-telegram-bot
```
