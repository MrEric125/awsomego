# Milvus 向量数据库配置指南

## 概述

Milvus 是一个开源的向量数据库，专为海量向量数据的存储、索引和检索而设计。本指南将帮助您配置和使用 Milvus 作为 RAG 系统的向量存储。

## 安装 Milvus

### 方式一：使用 Docker（推荐）

```bash
# 下载 Milvus 配置文件
wget https://github.com/milvus-io/milvus/releases/download/v2.4.0/milvus-standalone-docker-compose.yml -O docker-compose.yml

# 启动 Milvus
docker-compose up -d

# 查看状态
docker-compose ps
```

### 方式二：使用 Docker Compose（完整版）

创建 `docker-compose.yml` 文件：

```yaml
version: '3.5'

services:
  etcd:
    container_name: milvus-etcd
    image: quay.io/coreos/etcd:v3.5.5
    environment:
      - ETCD_AUTO_COMPACTION_MODE=revision
      - ETCD_AUTO_COMPACTION_RETENTION=1000
      - ETCD_QUOTA_BACKEND_BYTES=4294967296
      - ETCD_SNAPSHOT_COUNT=50000
    volumes:
      - ${DOCKER_VOLUME_DIRECTORY:-.}/volumes/etcd:/etcd
    command: etcd -advertise-client-urls=http://127.0.0.1:2379 -listen-client-urls http://0.0.0.0:2379 --data-dir /etcd
    healthcheck:
      test: ["CMD", "etcdctl", "endpoint", "health"]
      interval: 30s
      timeout: 20s
      retries: 3

  minio:
    container_name: milvus-minio
    image: minio/minio:RELEASE.2023-03-20T20-16-18Z
    environment:
      MINIO_ACCESS_KEY: minioadmin
      MINIO_SECRET_KEY: minioadmin
    ports:
      - "9001:9001"
      - "9000:9000"
    volumes:
      - ${DOCKER_VOLUME_DIRECTORY:-.}/volumes/minio:/minio_data
    command: minio server /minio_data --console-address ":9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  standalone:
    container_name: milvus-standalone
    image: milvusdb/milvus:v2.4.0
    command: ["milvus", "run", "standalone"]
    security_opt:
      - seccomp:unconfined
    environment:
      ETCD_ENDPOINTS: etcd:2379
      MINIO_ADDRESS: minio:9000
    volumes:
      - ${DOCKER_VOLUME_DIRECTORY:-.}/volumes/milvus:/var/lib/milvus
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9091/healthz"]
      interval: 30s
      start_period: 90s
      timeout: 20s
      retries: 3
    ports:
      - "19530:19530"
      - "9091:9091"
    depends_on:
      - "etcd"
      - "minio"

networks:
  default:
    name: milvus
```

启动服务：

```bash
docker-compose up -d
```

### 方式三：使用 Kubernetes

参考官方文档：https://milvus.io/docs/install_cluster-helm.md

## 配置环境变量

在项目根目录创建 `.env` 文件：

```bash
# 向量数据库类型
VECTOR_STORE_TYPE=milvus

# Milvus 配置
MILVUS_HOST=localhost
MILVUS_PORT=19530
MILVUS_COLLECTION=rag_knowledge_base

# 向量维度（根据 Embedding 模型调整）
VECTOR_DIMENSION=1536

# Embedding 配置
EMBEDDING_MODEL=text-embedding-ada-002
EMBEDDING_API_KEY=your-openai-api-key
EMBEDDING_BASE_URL=https://api.openai.com/v1

# RAG 配置
RAG_TOP_K=5
RAG_SCORE_THRESHOLD=0.7
CHUNK_SIZE=1000
CHUNK_OVERLAP=200
```

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
    // 加载配置
    aiCfg := config.LoadAIConfig()
    ragCfg := ragConfig.LoadRAGConfig()

    // 配置使用 Milvus
    ragCfg.VectorStoreType = "milvus"

    // 创建聊天模型
    chatModelProvider := components.NewChatModelProvider(aiCfg)
    chatModel, err := chatModelProvider.GetDefaultChatModel()
    if err != nil {
        log.Fatalf("Failed to create chat model: %v", err)
    }

    // 创建 Embedding 模型
    embeddingProvider := ragComponents.NewEmbeddingProvider(ragCfg)
    embeddingModel, err := embeddingProvider.GetEmbeddingModel()
    if err != nil {
        log.Fatalf("Failed to get embedding model: %v", err)
    }

    // 创建 Milvus 向量存储
    vectorStoreProvider := ragComponents.NewVectorStoreProvider(ragCfg, embeddingModel)
    vectorStore, err := vectorStoreProvider.GetVectorStore()
    if err != nil {
        log.Fatalf("Failed to create Milvus vector store: %v", err)
    }

    // 创建 RAG 服务
    ragSvc := ragService.NewRAGService(chatModel, vectorStore, ragCfg)

    ctx := context.Background()

    // 添加文档
    docs := []*ragService.Document{
        {
            Content: "Milvus 是一个开源的向量数据库。",
            MetaData: map[string]any{
                "source": "intro.txt",
            },
        },
    }
    ragSvc.AddDocuments(ctx, docs)

    // 查询
    response, err := ragSvc.Query(ctx, "什么是 Milvus？")
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    fmt.Printf("回答: %s\n", response.Answer)
}
```

### API 调用示例

```bash
# 添加文档
curl -X POST http://localhost:8080/api/rag/documents \
  -H "Content-Type: application/json" \
  -d '{
    "documents": [
      {
        "content": "Milvus 支持多种索引类型。",
        "metadata": {"source": "features.txt"}
      }
    ]
  }'

# 查询
curl -X POST http://localhost:8080/api/rag/query \
  -H "Content-Type: application/json" \
  -d '{"question": "Milvus 有哪些特性？"}'
```

## Milvus 管理工具

### Attu（Web UI）

Attu 是 Milvus 的图形化管理工具。

```bash
# 使用 Docker 启动 Attu
docker run -d \
  --name attu \
  -p 3000:3000 \
  -e MILVUS_URL=host.docker.internal:19530 \
  zilliz/attu:latest

# 访问 http://localhost:3000
```

### Birdwatcher（命令行工具）

```bash
# 安装
go install github.com/milvus-io/birdwatcher@latest

# 连接到 Milvus
birdwatcher --connect --addr localhost:19530
```

## 性能优化建议

### 1. 索引选择

Milvus 支持多种索引类型：

- **AUTOINDEX**: 自动选择最佳索引（推荐）
- **IVF_FLAT**: 适合中小规模数据
- **IVF_SQ8**: 节省内存，适合大规模数据
- **HNSW**: 高性能，适合实时搜索
- **ANNOY**: 适合高维向量

### 2. 集合配置

```go
// 自定义索引
idx, err := entity.NewIndexHNSW(entity.L2, 16, 256)
// 参数说明：
// - metricType: L2 或 IP（内积）
// - M: HNSW 的 M 参数（影响图连接度）
// - efConstruction: 构建时的 ef 参数
```

### 3. 搜索参数优化

```go
// 调整搜索参数
sp, err := entity.NewIndexHNSWSearchParam(64)
// ef 参数越大，搜索越精确，但速度越慢
```

### 4. 分区策略

对于大规模数据，建议使用分区：

```go
// 创建分区
err := client.CreatePartition(ctx, collectionName, partitionName)

// 插入数据到指定分区
client.Insert(ctx, collectionName, partitionName, columns...)

// 搜索指定分区
client.Search(ctx, collectionName, []string{partitionName}, ...)
```

## 监控和日志

### Prometheus 监控

Milvus 内置 Prometheus 指标：

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'milvus'
    static_configs:
      - targets: ['localhost:9091']
```

### Grafana 仪表板

导入 Milvus 官方 Grafana 仪表板：
https://grafana.com/grafana/dashboards/12018

## 故障排查

### 连接问题

```bash
# 检查 Milvus 是否运行
docker ps | grep milvus

# 检查端口
netstat -an | grep 19530

# 查看 Milvus 日志
docker logs milvus-standalone
```

### 性能问题

1. **检查索引是否创建**
   ```bash
   curl http://localhost:9091/api/v1/collection
   ```

2. **检查内存使用**
   ```bash
   docker stats milvus-standalone
   ```

3. **调整配置**
   - 增加 `cacheSize` 提高缓存
   - 调整 `searchConcurrency` 控制并发

## 备份和恢复

### 备份

```bash
# 使用 Milvus Backup 工具
milvus-backup create --name backup_20240101
```

### 恢复

```bash
milvus-backup restore --name backup_20240101
```

## 安全配置

### 启用认证

```yaml
# milvus.yaml
common:
  security:
    authorizationEnabled: true
```

### 创建用户

```bash
# 使用 Milvus CLI
connect -h localhost -p 19530
create user -u admin -p password123
```

## 集群部署

对于生产环境，建议使用集群模式：

```yaml
# docker-compose-cluster.yml
# 包含多个 query node、data node、index node
```

详细配置参考：https://milvus.io/docs/install_cluster-docker.md

## 常见问题

### Q: 如何选择向量维度？

A: 向量维度由 Embedding 模型决定：
- OpenAI text-embedding-ada-002: 1536
- Cohere embed-english-v3.0: 1024
- 自定义模型: 根据模型输出

### Q: 如何处理大规模数据？

A: 建议：
1. 使用分区按业务分割数据
2. 使用 HNSW 索引提高搜索速度
3. 增加节点数量进行水平扩展
4. 使用批量插入提高写入性能

### Q: 如何保证数据持久化？

A: Milvus 自动将数据持久化到 MinIO/S3。确保：
1. 配置持久化存储卷
2. 定期备份
3. 监控存储空间

## 参考资源

- [Milvus 官方文档](https://milvus.io/docs)
- [Milvus GitHub](https://github.com/milvus-io/milvus)
- [Attu GUI](https://github.com/zilliztech/attu)
- [Milvus Go SDK](https://github.com/milvus-io/milvus-sdk-go)
