package message

import (
	"duolaGPT/utils"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"

	"duolaGPT/conf"
	"duolaGPT/gptMessage"
	"duolaGPT/variables"
	"log"
	"strings"
	"sync"
)

var mu = &sync.Mutex{}
var FreeChatCount int

// UserManager 管理用户状态和消息计数
type UserManager struct {
	mu    sync.Mutex
	users map[int]*variables.User
}

// NewUserManager 创建UserManager的新实例
func NewUserManager() *UserManager {
	return &UserManager{
		users: make(map[int]*variables.User),
	}
}

func (manager *UserManager) IncrementMessageCount(userID int) int {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	user, exists := manager.users[userID]
	if !exists {
		user = &variables.User{} // 如果用户不存在，则创建一个新用户
		manager.users[userID] = user
	}
	user.MessageCount++
	return user.MessageCount
}

func (manager *UserManager) CheckUserAccess(config conf.Config, update tgbotapi.Update, bot *tgbotapi.BotAPI) bool {
	userID := update.Message.From.ID
	userName := update.Message.From.UserName

	// 如果用户在白名单中，直接返回true。
	if utils.StringInSlice(config.AllowedUsers, userName) {
		return true
	}

	// 如果用户不在白名单中，增加他们的消息计数。update.Message.From.ID 保证私聊和群组使用都被统计
	count := manager.IncrementMessageCount(int(userID))

	// 如果用户的消息计数超过FreeChatCount，通知用户并返回false。
	if count > FreeChatCount {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "体验对话次数已用尽.")
		bot.Send(msg)
		return false
	}
	fmt.Printf("当前用户%s-%d已对话%d次.", userName, userID, count)

	// 如果没有达到限制，返回true。
	return true
}

func HandleMessage(userManager *UserManager, config conf.Config, bot *tgbotapi.BotAPI, update tgbotapi.Update, client *openai.Client) {

	if !userManager.CheckUserAccess(config, update, bot) {
		return // 如果用户没有访问权限，则直接返回。
	}

	mu.Lock()

	user := variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID]
	state := user.State
	model := user.Model
	if model == "" {
		model = variables.DefaultModel
	}
	variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID] = variables.User{
		Model: model,
		State: state,
	}

	mu.Unlock()
	if state == variables.StateWaitingForSystemPrompt {
		mu.Lock()
		variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID] = variables.User{
			Model:        model,
			SystemPrompt: update.Message.Text,
			State:        variables.StateDefault,
		}
		mu.Unlock()
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "System prompt set.")
		bot.Send(msg)
		return
	}
	var err error
	generatedTextStream, err := gptMessage.GenerateTextStreamWithGPT(client, update.Message.Text, update.Message.From.ID+update.Message.Chat.ID, model)
	if err != nil {
		log.Printf("Failed to generate text stream with GPT: %v", err)
		return
	}
	var text string
	HasGetChangeID := false
	messageID := 0

	var charThreshold = 200
	var buffer strings.Builder

	for generatedText := range generatedTextStream {

		if HasGetChangeID == false {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "waiting...")
			msg.ReplyToMessageID = update.Message.MessageID
			msg_, err := bot.Send(msg)
			if err != nil {
				log.Printf("Failed to send message: %v", err)
			}
			messageID = msg_.MessageID
			HasGetChangeID = true
		}
		buffer.WriteString(generatedText)

		if buffer.Len() >= charThreshold {
			text = buffer.String()
			if len(text) > 4096 {
				if isCode(text) {
					// 使用 Markdown 格式化代码块
					formattedText := "```" + escapeMarkdownCode(text) + "```"
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, formattedText)
					msg.ParseMode = tgbotapi.ModeMarkdownV2 // 使用 Markdown V2
					msg_, err := bot.Send(msg)
					if err != nil {
						log.Printf("Failed to send message: %v", err)
					} else {
						messageID = msg_.MessageID
						buffer.Reset()
					}
				} else {
					// 发送普通文本
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
					msg_, err := bot.Send(msg)
					if err != nil {
						log.Printf("Failed to send message: %v", err)
					} else {
						messageID = msg_.MessageID
						buffer.Reset()
					}
				}

			} else {

				if buffer.Len() >= charThreshold {
					if isCode(text) {
						// 使用 Markdown 格式化代码块
						formattedText := "```" + escapeMarkdownCode(text) + "```"
						msg := tgbotapi.NewEditMessageText(update.Message.Chat.ID, messageID, formattedText)
						msg.ParseMode = tgbotapi.ModeMarkdownV2 // 使用 Markdown V2
						_, err := bot.Send(msg)
						if err != nil {
							log.Printf("Failed to send message: %v", err)
						}
						charThreshold += 100

					} else {
						// 发送普通文本
						msg := tgbotapi.NewEditMessageText(update.Message.Chat.ID, messageID, text)
						_, err := bot.Send(msg)
						if err != nil {
							log.Printf("Failed to send message: %v", err)
						}
						charThreshold += 100

					}

				}
			}
		}
	}
	if buffer.Len() > 0 {
		text = buffer.String()
		if len(text) > 4096 {
			if isCode(text) {
				// 使用 Markdown 格式化代码块
				formattedText := "```" + escapeMarkdownCode(text) + "```"
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, formattedText)
				msg.ParseMode = tgbotapi.ModeMarkdownV2 // 使用 Markdown V2
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Failed to send message: %v", err)
				}
			} else {
				// 发送普通文本
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Failed to send message: %v", err)
				}
			}

		} else {

			if isCode(text) {
				// 使用 Markdown 格式化代码块
				formattedText := "```" + escapeMarkdownCode(text) + "```"
				msg := tgbotapi.NewEditMessageText(update.Message.Chat.ID, messageID, formattedText)
				msg.ParseMode = tgbotapi.ModeMarkdownV2 // 使用 Markdown V2
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Failed to send message: %v", err)
				}
			} else {
				// 发送普通文本
				msg := tgbotapi.NewEditMessageText(update.Message.Chat.ID, messageID, text)
				_, err := bot.Send(msg)
				if err != nil {
					log.Printf("Failed to send message: %v", err)
				}
			}

		}
	}
	fmt.Printf("\n" + "#####################################################" + "\n")
	fmt.Printf("当前用户:%s-id:%d的对话历史:", update.Message.From.UserName, update.Message.From.ID)
	var t string
	for _, message := range variables.ConversationHistory[update.Message.From.ID+update.Message.Chat.ID] {
		t += message.Content + "-" // 累积对话历史
	}
	fmt.Println(t)
	fmt.Println("#####################################################")
	gptMessage.CompleteResponse(update.Message.From.ID + update.Message.Chat.ID)
}

func HandleImg(userManager *UserManager, config conf.Config, bot *tgbotapi.BotAPI, update tgbotapi.Update, client *openai.Client) {
	if !userManager.CheckUserAccess(config, update, bot) {
		return // 如果用户没有访问权限，则直接返回。
	}
	ImgArg := update.Message.CommandArguments()
	model := variables.GPTPICModel
	mu.Lock()
	variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID] = variables.User{
		Model:        variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID].Model,
		SystemPrompt: ImgArg,
	}
	mu.Unlock()
	waitingMsg, err := bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "waiting..."))
	if err != nil {
		log.Printf("Failed to send waiting message: %v", err)
		return
	}
	generatedImg, err := gptMessage.GenerateImgWithGPT(client, ImgArg, update.Message.Chat.ID, model)
	if err != nil {
		log.Printf("Failed to generate img with GPT: %v", err)
		deleteConfig := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, waitingMsg.MessageID)
		_, _ = bot.Request(deleteConfig)
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "当前 prompt 未能成功生成图片，可能因为版权，政治，色情，暴力，种族歧视等违反 OpenAI 的内容政策！"))
		return
	}
	// 删除"waiting..."消息
	deleteConfig := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, waitingMsg.MessageID)
	_, _ = bot.Request(deleteConfig)
	// 发送生成的图片
	bot.Send(generatedImg)
}

func HandleCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, client *openai.Client) {
	command := update.Message.Command()
	commandArg := update.Message.CommandArguments()
	// 获取用户ID
	userID := update.Message.From.ID + update.Message.Chat.ID

	// 检查用户是否已经有一个会话状态
	if _, exists := variables.UserSettingsMap[userID]; !exists {
		// 如果没有，则为该用户创建一个新的会话状态
		user := variables.UserSettingsMap[userID]
		variables.UserSettingsMap[userID] = user
	}

	switch command {
	case "start":
		mu.Lock()
		variables.ConversationHistory[update.Message.From.ID+update.Message.Chat.ID] = []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: variables.DefaultSystemPrompt,
			},
		}
		variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID] = variables.User{
			Model:        variables.DefaultModel,
			SystemPrompt: variables.DefaultSystemPrompt,
		}
		mu.Unlock()
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "欢迎来到哆啦助手!\n"+
			"/start - 清除 Prompt 和会话记录\n"+
			"/new - 仅清除会话记录\n"+
			"/gpt3 - 切换 GPT-3 模型\n"+
			"/gpt4 - 切换 GPT-4 模型\n"+
			"/pic - 切换图片模型\n"+
			"/stop - 中止 GPT 输出\n"+
			"/prompt - 设置 prompt 提示词")
		bot.Send(msg)
	case "new":
		mu.Lock()
		variables.ConversationHistory[update.Message.From.ID+update.Message.Chat.ID] = []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID].SystemPrompt,
			},
		}
		mu.Unlock()
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "已开启全新会话.")
		bot.Send(msg)
	case "gpt4":
		mu.Lock()
		variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID] = variables.User{
			Model: variables.GPT4Model,
		}
		mu.Unlock()
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "开启gpt-4-1106-preview模型.")
		bot.Send(msg)
	case "gpt3":
		mu.Lock()
		variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID] = variables.User{
			Model: variables.GPT35TurboModel,
		}
		mu.Unlock()
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "开启gpt-3.5-turbo模型.")
		bot.Send(msg)
	case "pic":
		mu.Lock()
		variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID] = variables.User{
			Model: variables.GPTPICModel,
		}
		mu.Unlock()
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "开启绘图. 使用方式: /pic 灰色的天空漫天的乌鸦")
		bot.Send(msg)
	case "stop":
		mu.Lock()
		user := variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID]
		(*user.CurrentContext)()
		user.CurrentContext = nil

		gptMessage.CompleteResponse(update.Message.From.ID + update.Message.Chat.ID)
		mu.Unlock()
	case "prompt":
		if commandArg == "" {
			mu.Lock()
			variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID] = variables.User{
				Model:        variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID].Model,
				SystemPrompt: variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID].SystemPrompt,
				State:        variables.StateWaitingForSystemPrompt,
			}
			mu.Unlock()
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "请输入你想要的prompt.")
			bot.Send(msg)
			return
		}
		mu.Lock()
		variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID] = variables.User{
			Model:        variables.UserSettingsMap[update.Message.From.ID+update.Message.Chat.ID].Model,
			SystemPrompt: commandArg,
		}
		variables.ConversationHistory[update.Message.From.ID+update.Message.Chat.ID] = []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: commandArg,
			},
		}
		mu.Unlock()
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("已设置自定义prompt: %s", commandArg))
		bot.Send(msg)
	default:
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("无效命令: %s", command))
		bot.Send(msg)
	}
}

func escapeMarkdownCode(text string) string {
	// 在 Markdown V2 中，以下字符需要在前面加上反斜杠进行转义
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, c := range specialChars {
		text = strings.ReplaceAll(text, c, "\\"+c)
	}
	return text
}

func isCode(text string) bool {
	// 一些常见的编程关键字
	keywords := []string{
		"class", "public", "private", "protected", "import", "package", // Java
		"func", "interface", "struct", "chan", "go", "select", // Go
		"def", "import", "class", "from", "self", "lambda", // Python
		"int", "char", "float", "double", "printf", "scanf", // C
		"cout", "cin", "namespace", "std", "nullptr", // C++
		"if", "else", "while", "for", "switch", "case", // Common to many languages
		"echo", "function", "if", "[", "]", // Shell
	}

	// 一些编程语言特有的字符或模式
	patterns := []string{
		"{", "}", "(", ")", ";", "[]", "#include", "//", "/*", "*/", "=>", // Common syntax elements
		"public static void main", // Java
		"fmt.", "go func",         // Go
		"def __init__",                  // Python
		"using namespace",               // C++
		"#!/bin/bash", "#!/usr/bin/env", // Shell shebang
	}

	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	// todo 这里可以添加更复杂的检测逻辑，例如正则表达式匹配等

	return false
}
