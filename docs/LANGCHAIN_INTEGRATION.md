# LangChain 集成指南

本项目已成功集成 **LangChain Go** (langchaingo)，与现有的 Eino 框架并存，为你提供多种 AI 开发选择。

## 📚 什么是 LangChain Go?

LangChain Go 是 Python 版 LangChain 的 Go 语言实现，专注于通过组合性来构建基于大型语言模型（LLMs）的应用程序。

### 主要特性

- ✅ **多模型支持**: OpenAI, Ollama（本地模型）, Azure OpenAI 等
- ✅ **链式编排**: LLMChain, ConversationChain, SummarizationChain 等
- ✅ **记忆管理**: 对话历史管理
- ✅ **文档问答**: RAG（检索增强生成）支持
- ✅ **Go 原生优势**: 高性能、并发安全、编译快

## 🏗️ 项目结构

```
internal/langchain/
├── config/              # 配置管理
│   └── config.go       # 从环境变量加载配置
├── components/          # 组件封装
│   └── llm_provider.go # LLM 提供者
├── service/             # 业务逻辑层
│   └── langchain_service.go # LangChain 服务实现
├── handlers/            # HTTP 处理器
│   └── langchain_handler.go # RESTful API 端点
└── init.go             # 模块初始化

examples/
└── langchain_example.go # 使用示例
```

## 🚀 快速开始

### 1. 配置环境变量

#### 使用 OpenAI（推荐）

```powershell
# Windows PowerShell
$env:OPENAI_API_KEY="your_openai_api_key_here"
$env:OPENAI_MODEL_NAME="gpt-3.5-turbo"
```

```bash
# Linux/Mac
export OPENAI_API_KEY="your_openai_api_key_here"
export OPENAI_MODEL_NAME="gpt-3.5-turbo"
```

#### 使用 Azure OpenAI（企业级）

```powershell
# Windows
$env:AZURE_OPENAI_API_KEY="your_azure_key"
$env:AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
$env:AZURE_OPENAI_DEPLOYMENT="gpt-35-turbo"
$env:LC_DEFAULT_PROVIDER="azure"
```

#### 使用 Anthropic Claude（长文本）

```powershell
# Windows
$env:ANTHROPIC_API_KEY="your_anthropic_key"
$env:ANTHROPIC_MODEL="claude-3-sonnet-20240229"
$env:LC_DEFAULT_PROVIDER="anthropic"
```

#### 使用 Google Gemini（多模态）

```powershell
# Windows
$env:GOOGLE_API_KEY="your_google_key"
$env:GOOGLE_MODEL="gemini-pro"
$env:LC_DEFAULT_PROVIDER="google"
```

#### 使用 DeepSeek（高性价比）

```powershell
# Windows
$env:DEEPSEEK_API_KEY="your_deepseek_key"
$env:DEEPSEEK_MODEL="deepseek-chat"
$env:LC_DEFAULT_PROVIDER="deepseek"
```

```bash
# Linux/Mac
export OPENAI_API_KEY="your_openai_api_key_here"
export OPENAI_MODEL_NAME="gpt-3.5-turbo"
```

#### 使用 Ollama（本地模型，免费）

首先安装 [Ollama](https://ollama.ai/)，然后运行：

```bash
ollama pull llama2
```

配置环境变量：

```powershell
# Windows
$env:OLLAMA_BASE_URL="http://localhost:11434"
$env:OLLAMA_MODEL="llama2"
```

```bash
# Linux/Mac
export OLLAMA_BASE_URL="http://localhost:11434"
export OLLAMA_MODEL="llama2"
```

#### 可选配置参数

```bash
# LangChain 通用配置
export LC_TEMPERATURE=0.7      # 温度参数 (0-2)
export LC_TOP_P=0.9            # Top-p 采样参数 (0-1)
export LC_MAX_TOKENS=2000      # 最大 token 数
export LC_TIMEOUT=30           # 超时时间（秒）
export LC_MAX_RETRIES=3        # 最大重试次数
```

### 2. 运行示例代码

```bash
go run examples/langchain_example.go
```

### 3. 启动 Web 服务

```bash
go run main.go
```

服务器将在 `:8080` 端口启动，LangChain API 端点在 `/api/lc` 路径下。

## 📡 API 端点总览

| 端点 | 方法 | 功能 | 请求体 |
|------|------|------|--------|
| `/api/lc/chat` | POST | 简单对话 | `{message, history?}` |
| `/api/lc/chain` | POST | 执行链 | `{prompt, chain_type}` |
| `/api/lc/summarize` | POST | 文本摘要 | `{text}` |
| `/api/lc/translate` | POST | 翻译 | `{text, target_lang}` |
| `/api/lc/qa` | POST | 文档问答 | `{question, documents}` |

### 1. 简单对话

**POST** `/api/lc/chat`

```json
{
  "message": "Go 语言的优势是什么？"
}
```

响应：
```json
{
  "reply": "Go 语言具有以下优势：...",
  "created": 1234567890
}
```

### 2. 带历史记录的对话

**POST** `/api/lc/chat`

```json
{
  "message": "它有什么优点？",
  "history": [
    {
      "role": "user",
      "content": "什么是微服务架构？"
    },
    {
      "role": "assistant",
      "content": "微服务架构是一种..."
    }
  ]
}
```

### 3. 执行不同类型的链

**POST** `/api/lc/chain`

```json
{
  "prompt": "列出三个著名的编程语言",
  "chain_type": "simple"
}
```

支持的链类型：
- `simple`: 简单 LLM 链
- `conversational`: 对话链（带记忆）
- `summarization`: 摘要链

### 4. 文本摘要

**POST** `/api/lc/summarize`

```json
{
  "text": "需要摘要的长文本..."
}
```

响应：
```json
{
  "summary": "简洁的摘要内容..."
}
```

### 5. 翻译

**POST** `/api/lc/translate`

```json
{
  "text": "Hello, World!",
  "target_lang": "中文"
}
```

响应：
```json
{
  "translation": "你好，世界！"
}
```

### 6. 文档问答（RAG）

**POST** `/api/lc/qa`

```json
{
  "question": "Go 语言是谁开发的？",
  "documents": [
    "Go 语言是由 Google 开发的开源编程语言。",
    "Go 语言的设计目标是简洁、高效和可靠。"
  ]
}
```

响应：
```json
{
  "answer": "Go 语言是由 Google 开发的。"
}
```

## 💻 代码示例

### 在代码中使用 LangChain 服务

```go
package main

import (
    "context"
    "awesome/internal/langchain/components"
    lcconfig "awesome/internal/langchain/config"
    "awesome/internal/langchain/service"
)

func main() {
    // 1. 加载配置
    cfg := lcconfig.LoadLangChainConfig()
    
    // 2. 创建提供者
    provider := components.NewLLMProvider(cfg)
    
    // 3. 创建服务
    lcService, err := service.NewLangChainService(provider)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // 4. 使用服务 - 简单对话
    reply, err := lcService.Chat(ctx, "你好！")
    if err != nil {
        panic(err)
    }
    
    println(reply)
    
    // 5. 使用服务 - 文本摘要
    summary, err := lcService.Summarize(ctx, longText)
    if err != nil {
        panic(err)
    }
    
    println(summary)
}
```

### 使用不同的链类型

```go
// 简单链
result, err := lcService.RunChain(ctx, prompt, service.SimpleChain)

// 对话链（带记忆）
result, err := lcService.RunChain(ctx, prompt, service.ConversationalChain)

// 摘要链
result, err := lcService.RunChain(ctx, text, service.SummarizationChain)
```

## 🔧 扩展其他模型提供商

目前支持 OpenAI 和 Ollama，可以轻松扩展支持其他提供商：

### 添加 Azure OpenAI 支持

在 `internal/langchain/components/llm_provider.go` 中添加：

```go
import "github.com/tmc/langchaingo/llms/azureopenai"

func (p *LLMProvider) GetAzureOpenAILLM() (llms.Model, error) {
    opts := []azureopenai.Option{
        azureopenai.WithToken(p.config.AzureAPIKey),
        azureopenai.WithModel(p.config.AzureModelName),
        azureopenai.WithBaseURL(p.config.AzureBaseURL),
    }
    
    llm, err := azureopenai.New(opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to create Azure OpenAI LLM: %w", err)
    }
    
    return llm, nil
}
```

## 🆚 Eino vs LangChain

项目同时集成了两个 AI 框架，你可以根据需求选择：

| 特性 | Eino | LangChain |
|------|------|-----------|
| 开发者 | 字节跳动 CloudWeGo | 社区驱动 |
| 模型支持 | 豆包 Ark, OpenAI | **6+ 提供商** (见下表) |
| 性能 | 高并发优化 | 标准 Go 性能 |
| 生态系统 | 字节内部验证 | 全球社区支持 |
| 适用场景 | 生产环境、高并发 | 快速原型、灵活性 |

### LangChain 支持的模型提供商

| 提供商 | 配置项 | 特点 | 推荐场景 |
|--------|---------|------|----------|
| **OpenAI** | `OPENAI_API_KEY` | GPT-4/3.5, 生态成熟 | 通用对话 |
| **Azure OpenAI** | `AZURE_OPENAI_*` | 企业级, SLA保障 | 企业应用 |
| **Anthropic Claude** | `ANTHROPIC_API_KEY` | 长文本, 逻辑推理强 | 文档分析 |
| **Google Gemini** | `GOOGLE_API_KEY` | 多模态能力强 | 图片/视频理解 |
| **DeepSeek** | `DEEPSEEK_API_KEY` | 高性价比, 中文好 | 成本敏感 |
| **Ollama** | 本地部署 | 隐私保护, 免费 | 本地开发 |

**建议**：
- 如果你使用豆包 Ark 或需要高并发性能 → 使用 **Eino**
- 如果你需要更多模型选择或本地模型 → 使用 **LangChain**
- 两个框架可以共存，根据场景灵活选择

## ⚠️ 注意事项

1. **API Key 安全**: 不要将 API Key 硬编码到代码中，使用环境变量或密钥管理服务
2. **成本控制**: 注意 API 调用费用，设置合理的 MaxTokens 和超时
3. **速率限制**: 遵守提供商的速率限制，实现适当的退避策略
4. **错误处理**: AI 调用可能失败，确保有完善的错误处理机制

## 🐛 故障排查

### 问题：OPENAI_API_KEY is not set

**解决方案**：确保已设置环境变量
```bash
# 检查是否设置
echo $OPENAI_API_KEY  # Linux/Mac
echo $env:OPENAI_API_KEY  # Windows PowerShell

# 设置环境变量
export OPENAI_API_KEY="your_key"  # Linux/Mac
$env:OPENAI_API_KEY="your_key"  # Windows PowerShell
```

### 问题：Ollama 连接失败

**解决方案**：
1. 确保 Ollama 已安装并运行：`ollama serve`
2. 检查模型已下载：`ollama pull llama2`
3. 验证 URL 是否正确：`curl http://localhost:11434/api/tags`

### 问题：API 返回错误

**解决方案**：
1. 检查 API Key 是否有效
2. 确认模型名称是否正确
3. 查看账户余额是否充足
4. 检查请求参数是否符合要求

## 📝 更新日志

- **v1.0.0** (2026-05-17): 初始版本
  - 集成 LangChain Go v0.1.14
  - 支持 OpenAI 和 Ollama 模型
  - 实现基础对话、链式编排、摘要、翻译、问答功能
  - 完整的 RESTful API
  - 与 Eino 框架共存

## 📚 参考资料

- [LangChain Go GitHub](https://github.com/tmc/langchaingo)
- [LangChain Go 文档](https://tmc.github.io/langchaingo/)
- [OpenAI API 文档](https://platform.openai.com/docs)
- [Ollama 官网](https://ollama.ai/)
