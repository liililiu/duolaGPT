package utils

import (
	"duolaGPT/conf"
	"duolaGPT/variables"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// GSearchResult 结构体用于解析Google Custom Search API的响应
type GSearchResult struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

func PerformGoogleSearch(apiKey, searchEngineID, searchQuery string, start, num int, language string, proxyURL string) (*GSearchResult, error) {
	baseURL := "https://www.googleapis.com/customsearch/v1"

	// 群聊截取对话文本
	if strings.Contains(searchQuery, "@") {
		parts := strings.Fields(searchQuery)
		if len(parts) > 1 {
			searchQuery = strings.Join(parts[1:], " ")
		}
	}
	searchURL := fmt.Sprintf("%s?key=%s&cx=%s&q=%s&start=%d&num=%d&lr=%s", baseURL, apiKey, searchEngineID, url.QueryEscape(searchQuery), start, num, language)

	var client *http.Client
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %v", err)
		}
		transport := &http.Transport{Proxy: http.ProxyURL(proxy)}
		client = &http.Client{Transport: transport}
	} else {
		client = &http.Client{}
	}

	resp, err := client.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	// 解析JSON响应
	var searchResult GSearchResult
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("json unmarshaling failed: %v", err)
	}

	return &searchResult, nil
}

func fetchURLContent(targetURL string, proxyURL string) (string, error) {
	client := &http.Client{}

	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse proxy URL: %v", err)
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	}
	resp, err := client.Get(targetURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
	}
	var sb strings.Builder
	_, err = io.Copy(&sb, resp.Body)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

func cleanText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func extractMainContent(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	var textContent strings.Builder

	// 查找所有的<p>标签并遍历
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		// 对于每个<p>标签，获取其文本内容，并追加到textContent中
		// 这里使用了一个换行符来分隔每个段落，根据需要可以移除或替换
		textContent.WriteString(s.Text() + "\n")
	})

	// 如果没有找到<p>标签，可能需要返回错误或空字符串
	if textContent.Len() == 0 {
		return "", fmt.Errorf("no <p> tags found")
	}

	// 过滤不符合预期的杂质内容(可能会筛选到无关的p标签)
	if len(cleanText(textContent.String())) < 200 {
		return "", nil
	}

	return cleanText(textContent.String()), nil
}
func ExtractSummariesFromSearchResult(searchResult *GSearchResult, proxyUrl string) []string {
	summaries := []string{}

	for _, item := range searchResult.Items {
		htmlContent, err := fetchURLContent(item.Link, proxyUrl)
		if err != nil {
			fmt.Printf("fetchURLContent error: %v\n", err)
			continue
		}

		mainContent, err := extractMainContent(htmlContent)
		if err != nil {
			fmt.Printf("extractMainContent error: %v\n", err)
			continue
		}

		//summary := fmt.Sprintf("Title: %s\nLink: %s\nContent: %s\n", item.Title, item.Link, mainContent)
		summary := fmt.Sprintf("%s\n", mainContent)
		summaries = append(summaries, summary)
	}

	return summaries
}

// CheckForKeywords 检查用户输入是否包含关键字
func CheckForKeywords(userInput string, config conf.Config) bool {
	if config.GoogleSearchEngineID == "" {
		return false
	}
	for _, keyword := range variables.TriggerKeywords {
		if strings.Contains(strings.ToLower(userInput), keyword) {
			return true
		}
	}
	return false
}
