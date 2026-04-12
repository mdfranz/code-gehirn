package summarizer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/mfranz/code-gehirn/internal/searcher"
	"github.com/mfranz/code-gehirn/internal/vault"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// Summarize answers query using context from the knowledge base.
//
// When vaultPath is non-empty the top-K matching files are read in full from
// disk, giving the LLM complete documents rather than 500-token chunks.
// When vaultPath is empty it falls back to the chunk-based RetrievalQA chain.
func Summarize(ctx context.Context, store qdrant.Store, llm llms.Model, query string, topK int, vaultPath string, maxTokens int) (string, error) {
	if vaultPath != "" {
		return summarizeFromFiles(ctx, store, llm, query, topK, vaultPath, maxTokens)
	}
	return summarizeFromChunks(ctx, store, llm, query, topK, maxTokens)
}

// summarizeFromFiles reads full source files discovered via Qdrant and feeds
// them as context to the LLM.
func summarizeFromFiles(ctx context.Context, store qdrant.Store, llm llms.Model, query string, topK int, vaultPath string, maxTokens int) (string, error) {
	results, err := searcher.Search(ctx, store, query, topK, 0)
	if err != nil {
		return "", fmt.Errorf("searching for relevant files: %w", err)
	}

	// Collect unique source files, preserving relevance order.
	seen := map[string]bool{}
	var filePaths []string
	for _, r := range results {
		if r.Path != "" && !seen[r.Path] {
			seen[r.Path] = true
			filePaths = append(filePaths, r.Path)
		}
	}

	if len(filePaths) == 0 {
		return "No relevant documents found.", nil
	}

	// Read full file contents from disk.
	var parts []string
	for _, rel := range filePaths {
		full, err := vault.ResolvePath(vaultPath, rel)
		if err != nil {
			slog.Warn("summarize: skipping unsafe path", "path", rel, "error", err)
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil {
			slog.Warn("summarize: skipping unreadable file", "path", full, "error", err)
			continue
		}
		parts = append(parts, fmt.Sprintf("### %s\n\n%s", rel, string(data)))
	}

	if len(parts) == 0 {
		return "Could not read any source files.", nil
	}

	context := strings.Join(parts, "\n\n---\n\n")
	prompt := fmt.Sprintf(
		"Use the following documents to provide a detailed answer or summary for the topic: %s\n\n"+
			"Documents:\n%s\n\n"+
			"If the information is not present in the documents, explain what is available or state that the specific topic isn't covered. "+
			"Focus on providing a helpful synthesis of the relevant parts.\n\n"+
			"Summary/Answer:",
		query, context,
	)

	slog.Info("llm summarize start (full files)", "query", query, "files", len(parts))
	start := time.Now()
	answer, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt, llms.WithMaxTokens(maxTokens))
	latency := time.Since(start).Milliseconds()
	if err != nil {
		slog.Error("llm summarize failed", "query", query, "latency_ms", latency, "error", err)
		return "", err
	}
	slog.Info("llm summarize done", "query", query, "files", len(parts), "latency_ms", latency)
	return answer, nil
}

// summarizeFromChunks is the original RetrievalQA chain approach used when no
// vault path is configured.
func summarizeFromChunks(ctx context.Context, store qdrant.Store, llm llms.Model, query string, topK int, maxTokens int) (string, error) {
	retriever := vectorstores.ToRetriever(store, topK)
	qaChain := chains.NewRetrievalQAFromLLM(llm, retriever)

	slog.Info("llm summarize start", "query", query, "top_k", topK)
	start := time.Now()
	res, err := chains.Call(ctx, qaChain, map[string]any{
		"query": query,
	}, chains.WithMaxTokens(maxTokens))
	latency := time.Since(start).Milliseconds()
	if err != nil {
		slog.Error("llm summarize failed", "query", query, "top_k", topK, "latency_ms", latency, "error", err)
		return "", err
	}
	slog.Info("llm summarize done", "query", query, "top_k", topK, "latency_ms", latency)

	if answer, ok := res["text"].(string); ok {
		return answer, nil
	}
	if answer, ok := res["answer"].(string); ok {
		return answer, nil
	}
	if answer, ok := res["output"].(string); ok {
		return answer, nil
	}
	return fmt.Sprintf("%v", res), nil
}
