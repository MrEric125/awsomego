# Eino AI Framework - Quick Start Guide

## 快速开始

### 1. 设置环境变量

#### Windows PowerShell
```powershell
# 设置豆包 Ark API Key
$env:ARK_API_KEY="your_api_key_here"
$env:ARK_MODEL_NAME="doubao-pro-32k"

# 可选：自定义配置
$env:AI_TIMEOUT=30
$env:AI_TEMPERATURE=0.7
```

#### Linux/Mac
```bash
# 设置豆包 Ark API Key
export ARK_API_KEY="your_api_key_here"
export ARK_MODEL_NAME="doubao-pro-32k"

# 可选：自定义配置
export AI_TIMEOUT=30
export AI_TEMPERATURE=0.7
```

### 2. 运行示例代码

```bash
# 运行 Eino 示例
go run examples/eino_example.go
```

### 3. 启动 Web 服务

```bash
# 启动服务器
go run main.go
```

服务器将在 `http://localhost:8080` 启动

### 4. 测试 API

#### 简单对话
```bash
curl -X POST http://localhost:8080/api/ai/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Go 语言的优势是什么？"}'
```

#### 流式对话
```bash
curl -X POST http://localhost:8080/api/ai/chat/stream \
  -H "Content-Type: application/json" \
  -N \
  -d '{"message": "介绍云计算的三个服务模式"}'
```

#### 文本摘要
```bash
curl -X POST http://localhost:8080/api/ai/summarize \
  -H "Content-Type: application/json" \
  -d '{"text": "需要摘要的长文本..."}'
```

#### 翻译
```bash
curl -X POST http://localhost:8080/api/ai/translate \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello, World!",
    "target_lang": "中文"
  }'
```

## 获取 API Key

### 豆包 Ark API Key

1. 访问 [火山引擎控制台](https://console.volcengine.com/)
2. 注册/登录账号
3. 进入「方舟」平台
4. 创建 API Key
5. 选择合适的模型（推荐：doubao-pro-32k）

详细文档：https://www.volcengine.com/docs/82379/1399008

## 项目结构

```
internal/ai/
├── config/          # 配置管理
├── components/      # Eino 组件封装
├── service/         # 业务逻辑层
├── handlers/        # HTTP 处理器
└── init.go         # 模块初始化

examples/
└── eino_example.go  # 使用示例

docs/
└── EINO_INTEGRATION.md  # 详细文档
```

## 常用命令

```bash
# 构建项目
go build ./...

# 运行主程序
go run main.go

# 运行示例
go run examples/eino_example.go

# 运行测试
go test ./...

# 格式化代码
go fmt ./...

# 整理依赖
go mod tidy
```

## 故障排查

### 问题：ARK_API_KEY is not set

**解决方案**：确保已正确设置环境变量
```powershell
# 检查环境变量
echo $env:ARK_API_KEY  # Windows
echo $ARK_API_KEY      # Linux/Mac
```

### 问题：连接超时

**解决方案**：
1. 检查网络连接
2. 增加超时时间：`export AI_TIMEOUT=60`
3. 确认 BASE_URL 正确

### 问题：API 返回错误

**解决方案**：
1. 验证 API Key 是否有效
2. 确认模型名称正确
3. 检查账户余额
4. 查看请求参数是否符合要求

## 更多资源

- 📖 [完整文档](docs/EINO_INTEGRATION.md)
- 🔗 [Eino 官方文档](https://www.cloudwego.io/zh/docs/eino/)
- 💻 [GitHub 仓库](https://github.com/cloudwego/eino)

## 支持的功能

✅ 简单对话  
✅ 带历史记录的对话  
✅ 流式对话（SSE）  
✅ 文本摘要  
✅ 翻译  
✅ 可扩展的组件架构  
✅ 支持多种模型提供商  

## 下一步

1. 阅读 [完整文档](docs/EINO_INTEGRATION.md) 了解更多功能
2. 尝试修改 `examples/eino_example.go` 进行实验
3. 在您的应用中集成 AI 功能
4. 探索 Eino 的高级特性（Graph 编排、Tool Calling、RAG 等）
