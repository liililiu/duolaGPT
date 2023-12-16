package variables

import (
	"context"
	"github.com/sashabaranov/go-openai"
)

var ConversationHistory = make(map[int64][]openai.ChatCompletionMessage)
var UserSettingsMap = make(map[int64]User)

const (
	GPT4Model                   = "gpt-4-1106-preview"
	GPT35TurboModel             = "gpt-3.5-turbo-1106"
	GPTPICModel                 = "dall-e-3"
	StateDefault                = ""
	StateWaitingForSystemPrompt = "waiting_for_system_prompt"
	DefaultSystemPrompt         = "You are a helpful assistant."
	DefaultModel                = GPT35TurboModel
)

type User struct {
	Model                string
	SystemPrompt         string
	State                string
	CurrentContext       *context.CancelFunc
	CurrentMessageBuffer string
	MessageCount         int
}
