package components

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"awesome/internal/ai/rag/config"

	"github.com/cloudwego/eino/schema"
)

// DocumentLoader 文档加载器接口
type DocumentLoader interface {
	Load(ctx context.Context) ([]*schema.Document, error)
}

// TextLoader 文本加载器
type TextLoader struct {
	text     string
	metadata map[string]any
}

// NewTextLoader 创建文本加载器
func NewTextLoader(text string, metadata map[string]any) *TextLoader {
	return &TextLoader{
		text:     text,
		metadata: metadata,
	}
}

// Load 加载文本
func (l *TextLoader) Load(ctx context.Context) ([]*schema.Document, error) {
	return []*schema.Document{
		{
			Content:  l.text,
			MetaData: l.metadata,
		},
	}, nil
}

// TextSplitter 文本分割器
type TextSplitter struct {
	chunkSize    int
	chunkOverlap int
	separators   []string
}

// NewTextSplitter 创建文本分割器
func NewTextSplitter(cfg *config.RAGConfig) *TextSplitter {
	return &TextSplitter{
		chunkSize:    cfg.ChunkSize,
		chunkOverlap: cfg.ChunkOverlap,
		separators:   []string{"\n\n", "\n", "。", "！", "？", ".", "!", "?", " ", ""},
	}
}

// SplitText 分割文本
func (s *TextSplitter) SplitText(text string) []string {
	// 首先按段落分割
	paragraphs := strings.Split(text, "\n\n")

	var chunks []string
	var currentChunk strings.Builder
	currentLength := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// 如果当前段落就超过 chunkSize，需要进一步分割
		if len(para) > s.chunkSize {
			// 先保存当前 chunk
			if currentLength > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentLength = 0
			}

			// 分割大段落
			subChunks := s.splitLargeText(para)
			chunks = append(chunks, subChunks...)
			continue
		}

		// 检查是否需要开始新的 chunk
		if currentLength+len(para) > s.chunkSize {
			if currentLength > 0 {
				chunks = append(chunks, currentChunk.String())

				// 处理重叠
				overlapText := s.getOverlapText(currentChunk.String())
				currentChunk.Reset()
				currentChunk.WriteString(overlapText)
				currentLength = len(overlapText)
			}
		}

		if currentLength > 0 {
			currentChunk.WriteString("\n\n")
			currentLength += 2
		}
		currentChunk.WriteString(para)
		currentLength += len(para)
	}

	// 添加最后一个 chunk
	if currentLength > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// splitLargeText 分割大文本块
func (s *TextSplitter) splitLargeText(text string) []string {
	var chunks []string
	var currentChunk strings.Builder
	currentLength := 0

	// 按句子分割
	sentences := s.splitSentences(text)

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// 如果单个句子就超过 chunkSize，强制分割
		if len(sentence) > s.chunkSize {
			if currentLength > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentLength = 0
			}

			// 强制按字符分割
			for i := 0; i < len(sentence); i += s.chunkSize {
				end := i + s.chunkSize
				if end > len(sentence) {
					end = len(sentence)
				}
				chunks = append(chunks, sentence[i:end])
			}
			continue
		}

		// 检查是否需要开始新的 chunk
		if currentLength+len(sentence) > s.chunkSize {
			if currentLength > 0 {
				chunks = append(chunks, currentChunk.String())

				// 处理重叠
				overlapText := s.getOverlapText(currentChunk.String())
				currentChunk.Reset()
				currentChunk.WriteString(overlapText)
				currentLength = len(overlapText)
			}
		}

		currentChunk.WriteString(sentence)
		currentLength += len(sentence)
	}

	if currentLength > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// splitSentences 分割句子
func (s *TextSplitter) splitSentences(text string) []string {
	// 中文和英文句子分割
	re := regexp.MustCompile(`[^。！？.!?]+[。！？.!?]+`)
	matches := re.FindAllString(text, -1)

	// 处理没有匹配到的情况
	if len(matches) == 0 {
		return []string{text}
	}

	// 检查是否有剩余文本
	lastIndex := 0
	for _, match := range matches {
		idx := strings.Index(text[lastIndex:], match)
		lastIndex += idx + len(match)
	}

	if lastIndex < len(text) {
		matches = append(matches, text[lastIndex:])
	}

	return matches
}

// getOverlapText 获取重叠文本
func (s *TextSplitter) getOverlapText(text string) string {
	if len(text) <= s.chunkOverlap {
		return text
	}

	// 从后往前找合适的分割点
	overlapText := text[len(text)-s.chunkOverlap:]

	// 尝试在句子边界分割
	for _, sep := range s.separators {
		if idx := strings.LastIndex(overlapText, sep); idx > 0 {
			return overlapText[idx+len(sep):]
		}
	}

	return overlapText
}

// SplitDocuments 分割文档
func (s *TextSplitter) SplitDocuments(docs []*schema.Document) []*schema.Document {
	var result []*schema.Document

	for _, doc := range docs {
		chunks := s.SplitText(doc.Content)

		for i, chunk := range chunks {
			// 复制元数据
			metadata := make(map[string]any)
			for k, v := range doc.MetaData {
				metadata[k] = v
			}
			metadata["chunk_index"] = i
			metadata["total_chunks"] = len(chunks)

			result = append(result, &schema.Document{
				Content: chunk,
				MetaData:    metadata,
			})
		}
	}

	return result
}

// DocumentProcessor 文档处理器
type DocumentProcessor struct {
	splitter *TextSplitter
}

// NewDocumentProcessor 创建文档处理器
func NewDocumentProcessor(cfg *config.RAGConfig) *DocumentProcessor {
	return &DocumentProcessor{
		splitter: NewTextSplitter(cfg),
	}
}

// Process 处理文档（加载并分割）
func (p *DocumentProcessor) Process(loader DocumentLoader) ([]*schema.Document, error) {
	docs, err := loader.Load(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load documents: %w", err)
	}

	return p.splitter.SplitDocuments(docs), nil
}

// SplitDocuments 分割文档列表
func (p *DocumentProcessor) SplitDocuments(docs []*schema.Document) []*schema.Document {
	return p.splitter.SplitDocuments(docs)
}
