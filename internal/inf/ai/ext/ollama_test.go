package ext

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/embedding/ollama"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/compose"
	callbacksHelper "github.com/cloudwego/eino/utils/callbacks"
	"github.com/ollama/ollama/api"
	"log"
	"net/http"
	"net/url"
	"testing"

	"os"
	"time"
)

func TestOllama(t *testing.T) {
	ctx := context.Background()

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434" // 默认本地
	}
	model := os.Getenv("OLLAMA_EMBED_MODEL")
	if model == "" {
		model = "qwen2.5-coder:7b"
	}

	embedder, err := ollama.NewEmbedder(ctx, &ollama.EmbeddingConfig{
		BaseURL: baseURL,
		Model:   model,
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("NewEmbedder of ollama error: %v", err)
		return
	}

	log.Printf("===== call Embedder directly =====")

	vectors, err := embedder.EmbedStrings(ctx, []string{"hello", "how are you"})
	if err != nil {
		log.Fatalf("EmbedStrings of Ollama failed, err=%v", err)
	}

	log.Printf("vectors : %v", vectors)

	log.Printf("===== call Embedder in Chain =====")

	handlerHelper := &callbacksHelper.EmbeddingCallbackHandler{
		OnStart: func(ctx context.Context, runInfo *callbacks.RunInfo, input *embedding.CallbackInput) context.Context {
			log.Printf("input access, len: %v, content: %s\n", len(input.Texts), input.Texts)
			return ctx
		},
		OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *embedding.CallbackOutput) context.Context {
			log.Printf("output finished, len: %v\n", len(output.Embeddings))
			return ctx
		},
	}

	handler := callbacksHelper.NewHandlerHelper().
		Embedding(handlerHelper).
		Handler()

	chain := compose.NewChain[[]string, [][]float64]()
	chain.AppendEmbedding(embedder)

	// 编译并运行
	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatalf("chain Compile failed, err=%v", err)
	}

	vectors, err = runnable.Invoke(ctx, []string{"hello", "how are you"},
		compose.WithCallbacks(handler))
	if err != nil {
		log.Fatalf("Invoke of runnable failed, err=%v", err)
	}

	log.Printf("vectors in chain: %v", vectors)
}

func TestOriginOllama(t *testing.T) {
	ctx := context.Background()

	ollamaClient, err := NewEmbedder(ctx, &ollama.EmbeddingConfig{
		BaseURL: "http://localhost:11434",
		Model:   "qwen2.5-coder:7b",
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("NewEmbedder of ollama error: %v", err)
		return
	}
	messages := []api.Message{
		api.Message{
			Role:    "system",
			Content: "Provide very brief, concise responses",
		},
		api.Message{
			Role:    "user",
			Content: "Name some unusual animals",
		},
		api.Message{
			Role:    "assistant",
			Content: "Monotreme, platypus, echidna",
		},
		api.Message{
			Role:    "user",
			Content: "which of these is the most dangerous?",
		},
	}

	req := &api.ChatRequest{
		Model:    "qwen2.5-coder:7b",
		Messages: messages,
	}

	respFunc := func(resp api.ChatResponse) error {
		fmt.Printf("ollama client chat response: %s\n", resp.Message.Content)
		return nil
	}
	err = ollamaClient.Chat(ctx, req, respFunc)
	if err != nil {
		log.Fatal(err)
	}
	//
	//generatorReq:=&api.GenerateRequest{
	//	Model: "qwen2.5-coder:7b",
	//	Messages: messages,
	//}
	//
	//ollamaClient.Generate(ctx,generatorReq,respFunc)

}

func NewEmbedder(ctx context.Context, config *ollama.EmbeddingConfig) (*api.Client, error) {
	if config == nil {
		return nil, fmt.Errorf("embedding config must not be nil")
	}

	var httpClient *http.Client
	if config.HTTPClient != nil {
		httpClient = config.HTTPClient
	} else {
		httpClient = &http.Client{Timeout: config.Timeout}
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	cli := api.NewClient(baseURL, httpClient)
	return cli, nil
}
