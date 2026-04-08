package summarizer

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// Summarize retrieves the top-K documents most similar to query, then feeds
// them into a RetrievalQA chain using the provided LLM.
func Summarize(ctx context.Context, store qdrant.Store, llm llms.Model, query string, topK int) (string, error) {
	retriever := vectorstores.ToRetriever(store, topK)
	qaChain := chains.NewRetrievalQAFromLLM(llm, retriever)

	res, err := chains.Call(ctx, qaChain, map[string]any{
		"query": query,
	}, chains.WithMaxTokens(1024))
	if err != nil {
		return "", err
	}

	answer, ok := res["text"].(string)
	if !ok {
		// Fallback to other common output keys if "text" is not present
		if a, ok := res["answer"].(string); ok {
			return a, nil
		}
		if a, ok := res["output"].(string); ok {
			return a, nil
		}
		return fmt.Sprintf("%v", res), nil
	}
	return answer, nil
}
