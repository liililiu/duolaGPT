package conf

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	ProxyUrl             string   `yaml:"proxy_url"`
	BaseUrl              string   `yaml:"base_url"`
	TelegramToken        string   `yaml:"telegram_token"`
	OpenAIKey            string   `yaml:"openai_api_key"`
	Temperature          float32  `yaml:"temperature"`
	AllowedUsers         []string `yaml:"allowed_telegram_usernames"`
	FreeChatCount        int      `yaml:"free_chat_count"`
	GoogleSearchKey      string   `yaml:"google_search_key"`
	GoogleSearchEngineID string   `yaml:"google_search_engine_id"`
}

func ReadConfig() (Config, error) {
	var config Config
	configFile, err := os.Open("config.yml")
	if err != nil {
		return config, err
	}
	defer configFile.Close()
	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}
