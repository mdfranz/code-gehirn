package summarizer

import (
	"context"

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

	answer, err := chains.Run(ctx, qaChain, query,
		chains.WithMaxTokens(1024),
	)
	if err != nil {
		return "", err
	}
	return answer, nil
}
