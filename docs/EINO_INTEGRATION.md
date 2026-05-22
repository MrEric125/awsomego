# Eino AI 框架集成指南

本项目已成功集成字节跳动 CloudWeGo 团队开源的 **Eino** AI 应用开发框架。

## 📚 什么是 Eino？

Eino 是一个基于 Go 语言的大模型应用开发框架，类似于 Python 的 LangChain，但具有以下优势：

- ✅ **强类型安全**：编译时检查，减少运行时错误
- ✅ **高并发性能**：充分利用 Go 的并发优势
- ✅ **组件化设计**：模块化、可编排、易扩展
- ✅ **生产级稳定**：经过字节跳动内部业务验证

## 🏗️ 项目结构

```
internal/ai/
├── config/              # AI 配置管理
│   └── config.go       # 从环境变量加载配置
├── components/          # Eino 组件封装
│   └── chat_model.go   # ChatModel 提供者
├── service/             # AI 业务逻辑层
│   └── ai_service.go   # AI 服务实现
├── handlers/            # HTTP 处理器
│   └── ai_handler.go   # AI API 端点
└── init.go             # 模块初始化

examples/
└── eino_example.go     # 使用示例
```

## 🚀 快速开始

### 1. 配置环境变量

在使用 AI 功能之前，需要配置大模型的 API Key。目前支持豆包 Ark（默认）和 OpenAI。

#### 豆包 Ark 配置（推荐）

```bash
# Windows PowerShell
$env:ARK_API_KEY="your_ark_api_key_here"
$env:ARK_MODEL_NAME="doubao-pro-32k"
$env:ARK_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"

# Linux/Mac
export ARK_API_KEY="your_ark_api_key_here"
export ARK_MODEL_NAME="doubao-pro-32k"
export ARK_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"
```

#### 可选配置参数

```bash
# 通用 AI 配置
export AI_TIMEOUT=30              # 超时时间（秒）
export AI_MAX_RETRIES=3           # 最大重试次数
export AI_TEMPERATURE=0.7         # 温度参数 (0-1)
export AI_TOP_P=0.9               # Top-p 采样参数
export AI_MAX_TOKENS=2000         # 最大 token 数
```

### 2. 运行示例代码

```bash
# 设置环境变量后运行示例
go run examples/eino_example.go
```

### 3. 启动 Web 服务

```bash
# 启动服务器
go run main.go

# 服务器将在 :8080 端口启动
```

## 📡 API 端点

启动服务后，可以使用以下 API 端点：

### 1. 简单对话

**POST** `/api/ai/chat`

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

**POST** `/api/ai/chat`

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

### 3. 流式对话（SSE）

**POST** `/api/ai/chat/stream`

```json
{
  "message": "请介绍一下云计算"
}
```

响应为 Server-Sent Events 流：
```
event: message
data: 云

event: message
data: 计算

...
```

### 4. 文本摘要

**POST** `/api/ai/summarize`

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

**POST** `/api/ai/translate`

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

## 💻 代码示例

### 在代码中使用 AI 服务

```go
package main

import (
    "context"
    "awesome/internal/ai/components"
    aiconfig "awesome/internal/ai/config"
    "awesome/internal/ai/service"
)

func main() {
    // 1. 加载配置
    cfg := aiconfig.LoadAIConfig()
    
    // 2. 创建提供者
    provider := components.NewChatModelProvider(cfg)
    
    // 3. 创建服务
    aiService, err := service.NewAIService(provider)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // 4. 使用服务
    reply, err := aiService.Chat(ctx, "你好！")
    if err != nil {
        panic(err)
    }
    
    println(reply)
}
```

### 流式对话示例

```go
// 流式对话
stream, err := aiService.StreamChat(ctx, "介绍 Go 语言")
if err != nil {
    panic(err)
}

for chunk := range stream {
    print(chunk) // 实时输出
}
```

## 🔧 扩展其他模型提供商

目前默认支持豆包 Ark，可以轻松扩展支持其他提供商：

### 添加 OpenAI 支持

在 `internal/ai/components/chat_model.go` 中添加：

```go
import "github.com/cloudwego/eino-ext/components/model/openai"

func (p *ChatModelProvider) GetOpenAIChatModel() (model.ChatModel, error) {
    chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatConfig{
        Model:  p.config.OpenAIModelName,
        APIKey: p.config.OpenAIAPIKey,
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to create OpenAI chat model: %w", err)
    }
    
    return chatModel, nil
}
```

## 🎯 核心特性

### 1. 组件化架构

- **ChatModel**: 统一的大模型调用接口
- **Tool**: 工具调用能力（Function Calling）
- **Retriever**: 向量检索（RAG）
- **ChatTemplate**: Prompt 模板引擎

### 2. 编排能力

Eino 支持 Chain 和 Graph 两种编排方式：

```go
// Chain 链式编排
chain := NewChain().
    Append(chatTemplate).
    Append(chatModel).
    Append(outputParser)

// Graph 图编排（支持复杂流程）
graph := NewGraph()
graph.AddNode("retriever", retriever)
graph.AddNode("chat", chatModel)
graph.AddEdge("retriever", "chat")
```

### 3. 流式处理

内置强大的流处理能力，自动处理流的拼接、拆分、复制和合并。

### 4. 可观测性

通过 Callback 机制实现日志、追踪、监控：

```go
type MyCallback struct{}

func (c *MyCallback) OnStart(ctx context.Context, info *RunInfo, input CallbackInput) {
    log.Printf("Start: %s", info.Name)
}

func (c *MyCallback) OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) {
    log.Printf("End: %s", info.Name)
}
```

## 📖 更多资源

- [Eino 官方文档](https://www.cloudwego.io/zh/docs/eino/)
- [Eino GitHub 仓库](https://github.com/cloudwego/eino)
- [CloudWeGo 官网](https://www.cloudwego.io/)

## 🤝 最佳实践

1. **配置管理**: 使用环境变量管理敏感信息（API Key）
2. **错误处理**: 始终检查错误并适当处理
3. **超时控制**: 为 AI 调用设置合理的超时时间
4. **重试机制**: 对临时性错误实现重试逻辑
5. **缓存策略**: 对重复请求考虑使用缓存
6. **限流保护**: 避免超出 API 配额限制

## ⚠️ 注意事项

1. **API Key 安全**: 不要将 API Key 硬编码到代码中，使用环境变量或密钥管理服务
2. **成本控制**: 注意 API 调用费用，设置合理的 MaxTokens 和超时
3. **速率限制**: 遵守提供商的速率限制，实现适当的退避策略
4. **错误处理**: AI 调用可能失败，确保有完善的错误处理机制

## 🐛 故障排查

### 问题：ARK_API_KEY is not set

**解决方案**：确保已设置环境变量
```bash
# 检查是否设置
echo $ARK_API_KEY  # Linux/Mac
echo $env:ARK_API_KEY  # Windows PowerShell

# 设置环境变量
export ARK_API_KEY="your_key"  # Linux/Mac
$env:ARK_API_KEY="your_key"  # Windows PowerShell
```

### 问题：连接超时

**解决方案**：
1. 检查网络连接
2. 增加超时时间：`export AI_TIMEOUT=60`
3. 检查 BASE_URL 是否正确

### 问题：API 返回错误

**解决方案**：
1. 检查 API Key 是否有效
2. 确认模型名称是否正确
3. 查看账户余额是否充足
4. 检查请求参数是否符合要求

## 📝 更新日志

- **v1.0.0** (2026-05-10): 初始版本
  - 集成 Eino v0.8.13
  - 支持豆包 Ark 模型
  - 实现基础对话、流式对话、摘要、翻译功能
  - 完整的 RESTful API
