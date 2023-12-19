package gptMessage

import (
	"bytes"
	"context"
	"duolaGPT/variables"
	"encoding/base64"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
	"image/png"
	"io"
	"log"
	"strings"
	"sync"
)

var mu sync.Mutex
var TemperatureNum float32

func GenerateTextStreamWithGPT(client *openai.Client, inputText string, chatID int64, model string) (chan string, error) {
	variables.ConversationHistory[chatID] = append(variables.ConversationHistory[chatID], openai.ChatCompletionMessage{
		Role:    "user",
		Content: inputText,
	})

	request := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    variables.ConversationHistory[chatID],
		Temperature: TemperatureNum,
		MaxTokens:   4096,
		TopP:        1,
		Stream:      true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	mu.Lock()
	user := variables.UserSettingsMap[chatID]
	user.CurrentContext = &cancel
	variables.UserSettingsMap[chatID] = user

	mu.Unlock()
	responseData := make(chan string)
	go func() {

		stream, err := client.CreateChatCompletionStream(ctx, request)
		if err != nil {
			fmt.Printf("ChatCompletionStream error: %v\n", err)
			return
		}
		defer stream.Close()
		for {
			select {
			case <-ctx.Done(): // 检查上下文是否被取消或超时
				fmt.Println("Context cancelled, closing stream!!!!!")
				return
			default:
				// 正常的流处理逻辑
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					fmt.Println("\nStream finished")
					return
				}

				if err != nil {
					fmt.Printf("\nStream error: %v\n", err)
					return
				}

				responseData <- response.Choices[0].Delta.Content

				mu.Lock()
				user = variables.UserSettingsMap[chatID]
				user.CurrentMessageBuffer += response.Choices[0].Delta.Content
				variables.UserSettingsMap[chatID] = user
				mu.Unlock()

				if response.Choices[0].FinishReason != "" {
					close(responseData)
					CompleteResponse(chatID)
				}
			}
		}

	}()

	return responseData, nil
}

func GenerateImgWithGPT(client *openai.Client, inputText string, chatID int64, model string) (tgbotapi.PhotoConfig, error) {
	ctx := context.Background()

	imageRequest := openai.ImageRequest{
		Model:          model,
		Prompt:         inputText,
		Size:           openai.CreateImageSize1024x1024,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		N:              1,
	}

	imageResponse, err := client.CreateImage(ctx, imageRequest)
	if err != nil {
		log.Printf("Failed to create image: %v", err)
		return tgbotapi.PhotoConfig{}, err
	}

	imageData, err := base64.StdEncoding.DecodeString(imageResponse.Data[0].B64JSON)
	if err != nil {
		log.Printf("Failed to decode base64 image data: %v", err)
		return tgbotapi.PhotoConfig{}, err
	}

	imageReader := bytes.NewReader(imageData)
	decodedImage, err := png.Decode(imageReader)
	if err != nil {
		log.Printf("Failed to decode PNG image: %v", err)
		return tgbotapi.PhotoConfig{}, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, decodedImage); err != nil {
		log.Printf("Failed to encode PNG image: %v", err)
		return tgbotapi.PhotoConfig{}, err
	}

	imageBufferReader := bytes.NewReader(buf.Bytes())

	imageFileReader := tgbotapi.FileReader{
		Name:   "image.png",
		Reader: imageBufferReader,
	}

	photoMessageConfig := tgbotapi.NewPhoto(chatID, imageFileReader)

	log.Println("The image was sent to the user.")
	return photoMessageConfig, nil
}

func CompleteResponse(chatID int64) {
	mu.Lock()
	user := variables.UserSettingsMap[chatID]
	generatedText := user.CurrentMessageBuffer
	user.CurrentMessageBuffer = ""
	variables.UserSettingsMap[chatID] = user
	mu.Unlock()
	generatedText = strings.TrimSpace(generatedText)
	variables.ConversationHistory[chatID] = append(variables.ConversationHistory[chatID], openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: generatedText,
	})
}
