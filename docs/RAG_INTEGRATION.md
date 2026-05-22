# RAG (Retrieval-Augmented Generation) 集成文档

## 概述

本项目已完整集成 RAG（检索增强生成）功能，支持基于知识库的智能问答。

## 架构组件

### 1. 配置层 (`internal/ai/rag/config`)
- **RAGConfig**: RAG 服务配置
  - Embedding 模型配置
  - 向量数据库配置
  - 检索参数配置
  - 文档分割配置

### 2. 组件层 (`internal/ai/rag/components`)

#### Embedding 模型 (`embedding.go`)
- **EmbeddingProvider**: Embedding 模型提供者
- **MockEmbedding**: Mock 实现（用于测试）
- 支持扩展：OpenAI、Azure、Cohere 等

#### 向量存储 (`vectorstore.go`)
- **VectorStore**: 向量存储接口
- **MemoryVectorStore**: 内存向量存储实现
- 支持扩展：Pinecone、Milvus、Chroma 等

#### 文档处理器 (`document_processor.go`)
- **DocumentLoader**: 文档加载器接口
- **TextLoader**: 文本加载器
- **TextSplitter**: 文本分割器
- **DocumentProcessor**: 文档处理器

#### 检索器 (`retriever.go`)
- **Retriever**: 检索器接口
- **RAGRetriever**: RAG 检索器
- **HybridRetriever**: 混合检索器（关键词 + 向量）

### 3. 服务层 (`internal/ai/rag/service`)
- **RAGService**: RAG 服务接口
- **RAGServiceImpl**: RAG 服务实现
  - 文档管理（添加、删除、清空）
  - 查询功能（普通查询、带历史查询、流式查询）

### 4. 处理器层 (`internal/ai/rag/handlers`)
- **RAGHandler**: HTTP 处理器
  - 文档管理 API
  - 查询 API
  - 统计信息 API

## API 接口

### 文档管理

#### 添加文档
```http
POST /api/rag/documents
Content-Type: application/json

{
  "documents": [
    {
      "content": "文档内容",
      "metadata": {
        "source": "example",
        "category": "test"
      }
    }
  ]
}
```

#### 添加文本
```http
POST /api/rag/text
Content-Type: application/json

{
  "text": "文本内容",
  "metadata": {
    "source": "example"
  }
}
```

#### 清空知识库
```http
DELETE /api/rag/clear
```

#### 获取统计信息
```http
GET /api/rag/stats
```

### 查询

#### 普通查询
```http
POST /api/rag/query
Content-Type: application/json

{
  "question": "什么是 Go 语言？"
}
```

响应：
```json
{
  "answer": "Go 语言是由 Google 开发的开源编程语言...",
  "sources": [
    {
      "content": "Go 语言是由 Google 开发的...",
      "metadata": {
        "source": "go_intro",
        "score": 0.95
      },
      "score": 0.95
    }
  ],
  "confidence": 0.95
}
```

#### 带历史记录的查询
```http
POST /api/rag/query/history
Content-Type: application/json

{
  "question": "它有什么特点？",
  "history": [
    {
      "role": "user",
      "content": "什么是 Go 语言？"
    },
    {
      "role": "assistant",
      "content": "Go 语言是由 Google 开发的..."
    }
  ]
}
```

#### 流式查询
```http
POST /api/rag/query/stream
Content-Type: application/json

{
  "question": "什么是 RAG？"
}
```

响应：Server-Sent Events (SSE) 流

## 配置说明

### 环境变量

```bash
# Embedding 配置
EMBEDDING_MODEL=text-embedding-ada-002
EMBEDDING_API_KEY=your_api_key
EMBEDDING_BASE_URL=https://api.openai.com/v1

# 向量数据库配置
VECTOR_STORE_TYPE=memory  # memory, pinecone, milvus, chroma
VECTOR_DIMENSION=1536

# 检索配置
RAG_TOP_K=5
RAG_SCORE_THRESHOLD=0.7

# 文档分割配置
CHUNK_SIZE=1000
CHUNK_OVERLAP=200
```

## 使用示例

### 代码示例

```go
package main

import (
    "context"
    "fmt"
    
    "awesome/internal/ai/rag/components"
    "awesome/internal/ai/rag/config"
    "awesome/internal/ai/rag/service"
)

func main() {
    // 1. 加载配置
    cfg := config.LoadRAGConfig()
    
    // 2. 创建 Embedding 模型
    embeddingProvider := components.NewEmbeddingProvider(cfg)
    embedder, _ := embeddingProvider.GetEmbeddingModel()
    
    // 3. 创建向量存储
    vectorStoreProvider := components.NewVectorStoreProvider(cfg, embedder)
    vectorStore, _ := vectorStoreProvider.GetVectorStore()
    
    // 4. 创建 RAG 服务
    ragService := service.NewRAGService(chatModel, vectorStore, cfg)
    
    // 5. 添加文档
    docs := []*service.Document{
        {
            Content: "Go 语言是由 Google 开发的开源编程语言。",
            Metadata: map[string]any{"source": "intro"},
        },
    }
    ragService.AddDocuments(context.Background(), docs)
    
    // 6. 查询
    response, _ := ragService.Query(context.Background(), "什么是 Go 语言？")
    fmt.Println(response.Answer)
}
```

### 运行示例

```bash
# 运行 RAG 示例
go run examples/rag_example.go
```

## 扩展指南

### 添加新的 Embedding 提供商

1. 在 `embedding.go` 中实现 `embedding.Embedder` 接口
2. 在 `EmbeddingProvider.GetEmbeddingModel()` 中添加新的提供商

### 添加新的向量存储

1. 在 `vectorstore.go` 中实现 `VectorStore` 接口
2. 在 `VectorStoreProvider.GetVectorStore()` 中添加新的存储类型

### 添加新的文档加载器

1. 在 `document_processor.go` 中实现 `DocumentLoader` 接口
2. 支持格式：PDF、Word、Markdown 等

## 性能优化

### 文档分割
- 调整 `CHUNK_SIZE` 和 `CHUNK_OVERLAP` 参数
- 根据文档类型选择合适的分割策略

### 检索优化
- 调整 `RAG_TOP_K` 控制检索文档数量
- 设置 `RAG_SCORE_THRESHOLD` 过滤低质量结果

### 向量存储
- 生产环境建议使用 Pinecone、Milvus 等专业向量数据库
- 内存存储适合开发和测试

## 最佳实践

1. **文档质量**: 确保添加的文档内容准确、完整
2. **元数据**: 为文档添加有意义的元数据，便于后续过滤
3. **分块策略**: 根据文档类型选择合适的分块大小
4. **检索调优**: 根据实际效果调整 TopK 和阈值参数
5. **监控**: 监控检索质量和生成效果，持续优化

## 故障排查

### 常见问题

1. **检索结果不准确**
   - 检查 Embedding 模型配置
   - 调整相似度阈值
   - 优化文档分块策略

2. **生成质量不佳**
   - 检查 ChatModel 配置
   - 优化提示词模板
   - 增加上下文文档数量

3. **性能问题**
   - 使用专业向量数据库
   - 优化文档分块大小
   - 考虑缓存机制

## 未来规划

- [ ] 支持更多 Embedding 提供商（OpenAI、Azure、Cohere）
- [ ] 集成 Pinecone、Milvus 向量数据库
- [ ] 支持更多文档格式（PDF、Word、Markdown）
- [ ] 实现混合检索（关键词 + 向量）
- [ ] 添加文档去重和更新功能
- [ ] 支持多租户和权限管理
- [ ] 添加检索结果重排序
