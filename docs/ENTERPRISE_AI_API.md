# Enterprise AI API 文档

## 概述

企业级 AI 服务提供 OpenAI 兼容的 API 接口，支持高并发、限流、重试、安全过滤和审计日志等企业级特性。

## 快速开始

### 环境配置

```bash
# OpenAI 配置
export OPENAI_API_KEY=your-api-key
export OPENAI_MODEL_NAME=gpt-4
export OPENAI_BASE_URL=https://api.openai.com/v1

# 通用配置
export AI_TIMEOUT=30
export AI_MAX_RETRIES=3
export AI_TEMPERATURE=0.7
export AI_MAX_TOKENS=2000
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    
    "awesome/internal/ai/config"
    "awesome/internal/ai/service"
    "awesome/internal/ai/openai"
)

func main() {
    // 加载配置
    cfg := config.LoadAIConfig()
    
    // 创建企业级服务
    svc, err := service.NewEnterpriseAIService(cfg,
        service.WithMaxConcurrency(100),
        service.WithRateLimit(1000, 100),
        service.WithCache(true, 10000, 5*time.Minute),
    )
    if err != nil {
        panic(err)
    }
    defer svc.Close()
    
    // 聊天请求
    resp, err := svc.Chat(context.Background(), &service.ChatRequest{
        Model: "gpt-4",
        Messages: []openai.ChatMessage{
            {Role: "user", Content: "Hello!"},
        },
    })
    
    fmt.Println(resp.Content)
}
```

## API 接口

### 1. 聊天完成

**POST** `/v1/chat/completions`

请求体:
```json
{
    "model": "gpt-4",
    "messages": [
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hello!"}
    ],
    "temperature": 0.7,
    "max_tokens": 2000,
    "use_cache": true
}
```

响应:
```json
{
    "id": "chatcmpl-xxx",
    "object": "chat.completion",
    "created": 1234567890,
    "model": "gpt-4",
    "choices": [
        {
            "index": 0,
            "message": {
                "role": "assistant",
                "content": "Hello! How can I help you today?"
            },
            "finish_reason": "stop"
        }
    ],
    "usage": {
        "prompt_tokens": 20,
        "completion_tokens": 10,
        "total_tokens": 30
    },
    "cached": false,
    "latency_ms": 1234
}
```

### 2. 流式聊天

**POST** `/v1/chat/completions/stream`

返回 Server-Sent Events (SSE) 格式的流式响应。

### 3. 创建嵌入

**POST** `/v1/embeddings`

请求体:
```json
{
    "model": "text-embedding-ada-002",
    "input": ["Hello world", "Goodbye world"]
}
```

响应:
```json
{
    "object": "list",
    "data": [
        {
            "object": "embedding",
            "index": 0,
            "embedding": [0.1, 0.2, ...]
        }
    ],
    "model": "text-embedding-ada-002",
    "usage": {
        "prompt_tokens": 4,
        "total_tokens": 4
    }
}
```

### 4. 健康检查

**GET** `/v1/health`

响应:
```json
{
    "status": "healthy",
    "timestamp": 1234567890,
    "service": "enterprise-ai"
}
```

### 5. 统计信息

**GET** `/v1/stats`

响应:
```json
{
    "total_requests": 1000,
    "success_count": 990,
    "failure_count": 10,
    "total_latency_ns": 1234567890,
    "client_stats": {
        "total_requests": 1000,
        "success_count": 990,
        "failure_count": 10,
        "circuit_state": "closed",
        "connection_pool": {
            "total_connections": 20,
            "active_connections": 5,
            "max_size": 20
        }
    }
}
```

## 企业级特性

### 1. 限流控制

支持多种限流算法:

- **令牌桶 (Token Bucket)**: 平滑限流，允许突发
- **滑动窗口 (Sliding Window)**: 精确限流
- **漏桶 (Leaky Bucket)**: 恒定速率

```go
// 配置限流
svc, _ := service.NewEnterpriseAIService(cfg,
    service.WithRateLimit(1000, 100), // 1000 QPS, burst 100
)
```

### 2. 重试机制

自动重试失败的请求，支持指数退避和抖动:

```go
// 重试策略配置
policy := retry.NewPolicy(
    retry.WithMaxRetries(3),
    retry.WithInitialDelay(100*time.Millisecond),
    retry.WithMaxDelay(5*time.Second),
    retry.WithMultiplier(2.0),
    retry.WithJitter(true),
)
```

### 3. 熔断器

防止级联故障，自动熔断和恢复:

```go
// 熔断器配置
breaker := NewCircuitBreaker(5, 30*time.Second)
// 5次失败后熔断，30秒后尝试恢复
```

### 4. 安全过滤

自动过滤敏感信息:

- API 密钥
- 密码
- 令牌
- 私钥
- 信用卡号
- 身份证号
- 手机号
- 邮箱

```go
// 使用安全过滤器
filter := security.NewFilter()
filtered, warnings := filter.Filter(content)
```

### 5. 审计日志

记录所有请求和响应:

```go
// 审计日志
audit := audit.NewLogger(logger)
audit.LogRequest(ctx, req)
audit.LogResponse(ctx, resp)
audit.LogSecurityEvent(ctx, "sensitive_content", details)
```

### 6. 性能监控

实时监控请求性能:

```go
// 获取指标
stats := client.GetMetrics()
fmt.Printf("成功率: %.2f%%\n", stats.SuccessRate*100)
fmt.Printf("平均延迟: %v\n", stats.AvgDuration)
```

### 7. 连接池

复用 HTTP 连接，提高性能:

```go
// 连接池配置
pool := NewConnectionPool(20) // 最大 20 个连接
conn := pool.Get()
defer pool.Put(conn)
```

### 8. 响应缓存

缓存重复请求的响应:

```go
// 启用缓存
svc, _ := service.NewEnterpriseAIService(cfg,
    service.WithCache(true, 10000, 5*time.Minute),
)

// 使用缓存
resp, _ := svc.Chat(ctx, &service.ChatRequest{
    Messages: messages,
    UseCache: true, // 启用缓存
})
```

## 错误处理

### 错误码

| 错误码 | 说明 | HTTP 状态码 |
|--------|------|-------------|
| `invalid_request` | 请求格式错误 | 400 |
| `unauthorized` | 认证失败 | 401 |
| `permission_denied` | 权限不足 | 403 |
| `model_not_found` | 模型不存在 | 404 |
| `rate_limit_exceeded` | 超过限流 | 429 |
| `server_error` | 服务器错误 | 500 |
| `service_unavailable` | 服务不可用 | 503 |
| `timeout` | 请求超时 | 504 |

### 错误响应格式

```json
{
    "error": {
        "code": "rate_limit_exceeded",
        "message": "Rate limit exceeded. Please retry after 60 seconds.",
        "type": "api_error"
    }
}
```

## 最佳实践

### 1. 并发控制

```go
// 设置最大并发数
svc, _ := service.NewEnterpriseAIService(cfg,
    service.WithMaxConcurrency(100),
)
```

### 2. 超时设置

```go
// 设置请求超时
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := svc.Chat(ctx, req)
```

### 3. 错误重试

```go
// 检查是否可重试
if errors.IsRetryable(err) {
    // 自动重试或手动重试
}
```

### 4. 优雅关闭

```go
// 关闭服务
defer svc.Close()
```

## 多语言调用示例

### Python

```python
import requests

response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    json={
        "model": "gpt-4",
        "messages": [
            {"role": "user", "content": "Hello!"}
        ]
    }
)

print(response.json())
```

### JavaScript/Node.js

```javascript
const response = await fetch('http://localhost:8080/v1/chat/completions', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
    },
    body: JSON.stringify({
        model: 'gpt-4',
        messages: [
            { role: 'user', content: 'Hello!' }
        ]
    })
});

const data = await response.json();
console.log(data);
```

### Java

```java
import java.net.http.*;
import java.net.URI;

HttpClient client = HttpClient.newHttpClient();
String body = """
    {
        "model": "gpt-4",
        "messages": [
            {"role": "user", "content": "Hello!"}
        ]
    }
    """;

HttpRequest request = HttpRequest.newBuilder()
    .uri(URI.create("http://localhost:8080/v1/chat/completions"))
    .header("Content-Type", "application/json")
    .POST(HttpRequest.BodyPublishers.ofString(body))
    .build();

HttpResponse<String> response = client.send(request, 
    HttpResponse.BodyHandlers.ofString());
System.out.println(response.body());
```

### cURL

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

## 监控和告警

### Prometheus 指标

```go
// 暴露 Prometheus 指标
http.Handle("/metrics", promhttp.Handler())
```

### 健康检查端点

```bash
# Kubernetes 探针
curl http://localhost:8080/v1/health
```

## 安全建议

1. **API 密钥管理**: 使用环境变量或密钥管理服务
2. **HTTPS**: 生产环境必须使用 HTTPS
3. **访问控制**: 实施适当的认证和授权
4. **日志脱敏**: 敏感数据自动脱敏
5. **审计追踪**: 记录所有操作日志

## 性能优化

1. **连接池**: 复用 HTTP 连接
2. **响应缓存**: 缓存重复请求
3. **并发控制**: 限制并发请求数
4. **流式响应**: 使用 SSE 流式传输
5. **批量请求**: 合并多个请求

## 故障排查

### 常见问题

1. **超时错误**: 增加 `AI_TIMEOUT` 配置
2. **限流错误**: 检查 `RateLimit` 配置
3. **连接错误**: 检查网络和代理设置
4. **认证错误**: 验证 API 密钥是否正确

### 日志级别

```bash
# 设置日志级别
export LOG_LEVEL=debug
```

## 版本历史

- **v1.0.0**: 初始版本
  - OpenAI API 兼容接口
  - 企业级特性支持
  - 多语言 SDK 支持
