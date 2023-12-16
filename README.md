# 哆啦助手gpt

哆啦助手gpt是一个基于Go语言开发，利用OpenAI开源模型提高生产效率的Telegram机器人。它集成了最新的人工智能技术，包括GPT-3和GPT-4对话模型，以及DALL-E 3图像生成模型，旨在为Telegram用户提供一个强大、灵活且高效的交流助手。

## 特点

- **支持GPT-3模型对话**: 利用GPT-3模型进行自然语言处理和生成对话。
- **支持GPT-4模型对话**: 接入最新的GPT-4模型，享受更加深入的对话体验。
- **支持DALL-E 3模型绘图**: 创造性地使用DALL-E 3模型生成图片。
- **上下文对话支持**: 保持对话连贯性，提供上下文相关的回答。
- **群聊功能**: 在群聊中使用，支持用户会话隔离，确保上下文不会混乱。
- **流式输出**: 优化输出体验，实时展示机器人回复。
- **自定义配置**: 支持自定义APIURL和OpenAI密钥。
- **代理支持**: 可配置代理以适应网络限制。
- **白名单模式**: 支持白名单模式，仅限授权用户使用。
- **markdown渲染输出**: 支持Markdown渲染，确保代码和文档的友好展示。

## 配置文件说明

```yaml
#proxy_url: "http://127.0.0.1:10809" # 可选的代理配置
base_url: "https://api.openai.com/v1" # API基础URL
openai_api_key: "sk-yourkey" # 你的OpenAI API密钥
temperature: 0.2 # 对话温度设置
telegram_token: "tg-yourtoken" # 你的Telegram机器人Token
allowed_telegram_usernames: ["tom","nick","tony"] # 允许使用机器人的Telegram用户名列表
free_chat_count: 10 # 免费对话次数限制
```

## 安装指南
要安装哆啦助手gpt，首先确保您的系统中已安装了Go语言环境。然后，按照以下步骤进行：

1. 克隆项目源码到本地：
   git clone https://github.com/liililiu/duolaGPT.git
2. 进入项目目录：
   cd duolaGPT
3. 按需编辑配置文件
4. 编译源代码：
   go build
5. 启动编译后的二进制文件即可运行哆啦助手gpt。

遵循以上步骤，您将能够在您的机器上成功部署和启动哆啦助手gpt

## 使用命令

- `/start` - 开启新对话，清除Prompt和会话记录。
- `/new` - 仅清除会话记录。
- `/gpt3` - 切换到GPT-3模型。
- `/gpt4` - 切换到GPT-4模型。
- `/pic` - 切换到图片生成模型。
- `/stop` - 中止GPT模型的输出。
- `/prompt` - 设置或更新会话的Prompt提示词。


## 试用机器人

立即体验哆啦助手gpt：[哆啦助手gpt](https://t.me/duolazhushou_bot)

