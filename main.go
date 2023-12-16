package main

import (
	"duolaGPT/conf"
	"duolaGPT/gptMessage"
	"duolaGPT/message"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	openai "github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func createHTTPClient(proxyURL string) *http.Client {
	if proxyURL == "" {
		return &http.Client{}
	}

	proxy, err := url.Parse(proxyURL)
	if err != nil {
		log.Fatalf("Failed to parse proxy URL: %v", err)
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}
}

func createOpenAIClient(msgConf conf.Config, httpClient *http.Client) *openai.Client {
	config := openai.DefaultConfig(msgConf.OpenAIKey)
	config.BaseURL = msgConf.BaseUrl
	config.HTTPClient = httpClient
	return openai.NewClientWithConfig(config)
}

func createTelegramBot(msgConf conf.Config, httpClient *http.Client) (*tgbotapi.BotAPI, error) {
	if msgConf.ProxyUrl != "" {
		return tgbotapi.NewBotAPIWithClient(msgConf.TelegramToken, "https://api.telegram.org/bot%s/%s", httpClient)
	}
	return tgbotapi.NewBotAPI(msgConf.TelegramToken)
}

func main() {
	msgConf, err := conf.ReadConfig()
	if err != nil {
		log.Fatalf("Failed to init msgConf: %v", err)
		return
	}

	gptMessage.TemperatureNum = msgConf.Temperature
	message.FreeChatCount = msgConf.FreeChatCount
	if msgConf.BaseUrl == "" {
		msgConf.BaseUrl = "https://openai.com/v1"
	}

	httpClient := createHTTPClient(msgConf.ProxyUrl)
	openAIClient := createOpenAIClient(msgConf, httpClient)

	bot, err := createTelegramBot(msgConf, httpClient)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
		return
	}
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("Failed to get updates channel: %v", err)
	}
	userManager := message.NewUserManager()
	for update := range updates {
		go func(update tgbotapi.Update) {

			if update.Message == nil {
				return
			}
			if update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup() {
				mentioned := false
				command := false
				for _, entity := range update.Message.Entities {
					if entity.Type == "mention" {
						// 提取提及的用户名
						username := update.Message.Text[entity.Offset : entity.Offset+entity.Length]
						if username == "@"+bot.Self.UserName {
							mentioned = true
							break
						}
					} else if entity.Type == "bot_command" {
						// 检查这是不是一个命令
						command = true
						break
					}
				}
				if !mentioned && !command {
					return
				}
				// 如果是命令，即使没有提及也处理
			}

			if update.Message.IsCommand() {
				cmd := update.Message.Command()
				cmdArgs := update.Message.CommandArguments()

				if cmd == "pic" && strings.TrimSpace(cmdArgs) != "" {
					message.HandleImg(userManager, msgConf, bot, update, openAIClient)
				} else if cmd == "pic" && strings.TrimSpace(cmdArgs) == "" {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "use the format: /pic 画一只小猫.")
					bot.Send(msg)
				} else {
					message.HandleCommand(bot, update, openAIClient)
				}
			} else {
				message.HandleMessage(userManager, msgConf, bot, update, openAIClient)
			}

		}(update)
	}
}
