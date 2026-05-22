# RAG (Retrieval-Augmented Generation) 模块

## 概述

RAG 模块为项目提供了完整的检索增强生成功能，支持将文档添加到知识库，并基于知识库进行智能问答。

## 功能特性

- **文档管理**: 支持添加文档和文本到知识库
- **智能检索**: 基于向量相似度的文档检索
- **上下文增强**: 自动将检索到的文档作为上下文提供给生成模型
- **流式响应**: 支持流式输出，提升用户体验
- **历史记录**: 支持带历史记录的对话
- **灵活配置**: 支持多种向量数据库和 Embedding 模型

## 架构组件

### 1. 配置层 (`config/`)
- `config.go`: RAG 配置管理，支持环境变量配置

### 2. 组件层 (`components/`)
- `embedding.go`: Embedding 模型提供者
- `vectorstore.go`: 向量存储实现（内存、Pinecone、Milvus 等）
- `document_loader.go`: 文档加载器和分割器
- `retriever.go`: 检索器实现

### 3. 服务层 (`service/`)
- `rag_service.go`: RAG 服务核心逻辑

### 4. 处理器层 (`handlers/`)
- `rag_handler.go`: HTTP API 处理器

## 配置说明

### 环境变量

```bash
# Embedding 配置
EMBEDDING_MODEL=text-embedding-ada-002
EMBEDDING_API_KEY=your-api-key
EMBEDDING_BASE_URL=https://api.openai.com/v1

# 向量数据库配置
VECTOR_STORE_TYPE=memory  # memory, pinecone, milvus, chroma
VECTOR_DIMENSION=1536

# Pinecone 配置（可选）
PINECONE_API_KEY=your-pinecone-key
PINECONE_ENVIRONMENT=your-environment
PINECONE_INDEX_NAME=your-index

# Milvus 配置（可选）
MILVUS_HOST=localhost
MILVUS_PORT=19530
MILVUS_COLLECTION=documents

# 检索配置
RAG_TOP_K=5
RAG_SCORE_THRESHOLD=0.7

# 文档分割配置
CHUNK_SIZE=1000
CHUNK_OVERLAP=200
```

## API 接口

### 1. 添加文档

**POST** `/api/rag/documents`

```json
{
  "documents": [
    {
      "content": "文档内容",
      "metadata": {
        "source": "document.txt",
        "type": "article"
      }
    }
  ]
}
```

### 2. 添加文本

**POST** `/api/rag/text`

```json
{
  "text": "文本内容",
  "metadata": {
    "source": "input"
  }
}
```

### 3. RAG 查询

**POST** `/api/rag/query`

```json
{
  "question": "什么是 RAG？"
}
```

**响应:**

```json
{
  "answer": "RAG 是一种结合检索和生成的 AI 技术...",
  "sources": [
    {
      "content": "检索到的文档内容",
      "metadata": {},
      "score": 0.95
    }
  ],
  "confidence": 0.95
}
```

### 4. 带历史记录的查询

**POST** `/api/rag/query/history`

```json
{
  "question": "它有哪些优势？",
  "history": [
    {
      "role": "user",
      "content": "什么是 RAG？"
    },
    {
      "role": "assistant",
      "content": "RAG 是一种结合检索和生成的 AI 技术..."
    }
  ]
}
```

### 5. 流式查询

**POST** `/api/rag/query/stream`

```json
{
  "question": "请详细解释 RAG 的工作原理"
}
```

**响应:** Server-Sent Events (SSE) 流

### 6. 清空知识库

**DELETE** `/api/rag/clear`

## 使用示例

### Go 代码示例

```go
package main

import (
    "context"
    "fmt"
    "log"

    "awesome/internal/ai/components"
    "awesome/internal/ai/config"
    ragComponents "awesome/internal/ai/rag/components"
    ragConfig "awesome/internal/ai/rag/config"
    ragService "awesome/internal/ai/rag/service"
)

func main() {
    // 1. 加载配置
    aiCfg := config.LoadAIConfig()
    ragCfg := ragConfig.LoadRAGConfig()

    // 2. 创建聊天模型
    chatModelProvider := components.NewChatModelProvider(aiCfg)
    chatModel, err := chatModelProvider.GetDefaultChatModel()
    if err != nil {
        log.Fatalf("Failed to create chat model: %v", err)
    }

    // 3. 创建 Embedding 模型
    embeddingProvider := ragComponents.NewEmbeddingProvider(ragCfg)
    embeddingModel, err := embeddingProvider.GetEmbeddingModel()
    if err != nil {
        log.Printf("Warning: Failed to get embedding model: %v", err)
        embeddingModel = &ragComponents.MockEmbedding{
            Dimension: ragCfg.VectorDimension,
        }
    }

    // 4. 创建向量存储
    vectorStoreProvider := ragComponents.NewVectorStoreProvider(ragCfg, embeddingModel)
    vectorStore, err := vectorStoreProvider.GetVectorStore()
    if err != nil {
        log.Fatalf("Failed to create vector store: %v", err)
    }

    // 5. 创建 RAG 服务
    ragSvc := ragService.NewRAGService(chatModel, vectorStore, ragCfg)

    ctx := context.Background()

    // 6. 添加文档
    docs := []*ragService.Document{
        {
            Content: "Go语言是由Google开发的一种静态强类型、编译型语言。",
            Metadata: map[string]any{
                "source": "go_intro.txt",
            },
        },
    }
    ragSvc.AddDocuments(ctx, docs)

    // 7. 查询
    response, err := ragSvc.Query(ctx, "什么是Go语言？")
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    fmt.Printf("回答: %s\n", response.Answer)
    fmt.Printf("置信度: %.2f\n", response.Confidence)
}
```

### cURL 示例

```bash
# 添加文档
curl -X POST http://localhost:8080/api/rag/documents \
  -H "Content-Type: application/json" \
  -d '{
    "documents": [
      {
        "content": "RAG是一种结合检索和生成的AI技术。",
        "metadata": {"source": "intro.txt"}
      }
    ]
  }'

# 查询
curl -X POST http://localhost:8080/api/rag/query \
  -H "Content-Type: application/json" \
  -d '{"question": "什么是RAG？"}'

# 流式查询
curl -X POST http://localhost:8080/api/rag/query/stream \
  -H "Content-Type: application/json" \
  -d '{"question": "请详细介绍RAG技术"}'
```

## 扩展开发

### 添加新的向量数据库

1. 在 `components/vectorstore.go` 中实现 `VectorStore` 接口
2. 在 `VectorStoreProvider.GetVectorStore()` 中添加新的 case

### 添加新的 Embedding 提供商

1. 在 `components/embedding.go` 中添加新的获取方法
2. 在 `GetEmbeddingModel()` 中添加优先级逻辑

### 自定义文档分割策略

1. 在 `components/document_loader.go` 中创建新的分割器
2. 实现 `SplitDocuments()` 方法

## 性能优化建议

1. **向量数据库选择**: 生产环境建议使用 Pinecone 或 Milvus
2. **Embedding 模型**: 使用高质量的 Embedding 模型提升检索准确度
3. **文档分割**: 根据文档类型调整 chunk_size 和 chunk_overlap
4. **缓存**: 对频繁查询的结果进行缓存
5. **批量处理**: 批量添加文档以提升性能

## 注意事项

1. 确保 `ARK_API_KEY` 或 `OPENAI_API_KEY` 环境变量已设置
2. 生产环境建议使用外部向量数据库而非内存存储
3. 文档分割参数需要根据实际文档类型调整
4. 流式查询需要客户端支持 SSE (Server-Sent Events)

## 许可证

MIT License
