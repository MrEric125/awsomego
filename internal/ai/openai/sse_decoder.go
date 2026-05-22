package openai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SSEDecoder SSE 解码器
type SSEDecoder struct {
	reader *bufio.Reader
}

// NewSSEDecoder 创建 SSE 解码器
func NewSSEDecoder(r io.Reader) *SSEDecoder {
	return &SSEDecoder{
		reader: bufio.NewReader(r),
	}
}

// Decode 解码下一个事件
func (d *SSEDecoder) Decode() (*StreamChunk, error) {
	for {
		line, err := d.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)

		// 跳过空行
		if line == "" {
			continue
		}

		// 解析 SSE 格式
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// 检查是否结束
			if data == "[DONE]" {
				return nil, io.EOF
			}

			// 解析 JSON
			var streamResp struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				Model   string `json:"model"`
				Choices []struct {
					Index        int `json:"index"`
					Delta        struct {
						Role    string `json:"role,omitempty"`
						Content string `json:"content,omitempty"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				return nil, fmt.Errorf("failed to parse SSE data: %w", err)
			}

			chunk := &StreamChunk{
				ID:      streamResp.ID,
				Object:  streamResp.Object,
				Created: streamResp.Created,
				Model:   streamResp.Model,
				Choices: make([]StreamChoice, len(streamResp.Choices)),
			}

			for i, choice := range streamResp.Choices {
				chunk.Choices[i] = StreamChoice{
					Index: choice.Index,
					Delta: ChatMessage{
						Role:    choice.Delta.Role,
						Content: choice.Delta.Content,
					},
					FinishReason: choice.FinishReason,
				}
			}

			return chunk, nil
		}
	}
}
