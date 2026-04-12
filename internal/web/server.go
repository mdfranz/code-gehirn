package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/mfranz/code-gehirn/internal/runtime"
	"github.com/mfranz/code-gehirn/internal/searcher"
	"github.com/mfranz/code-gehirn/internal/summarizer"
	"github.com/mfranz/code-gehirn/internal/vault"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	cfg   config.Config
	store qdrant.Store
	llm   llms.Model
}

func NewServer(cfg config.Config, store qdrant.Store, llm llms.Model) *Server {
	return &Server{
		cfg:   cfg,
		store: store,
		llm:   llm,
	}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// Static files
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// API endpoints
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/summarize", s.handleSummarize)
	mux.HandleFunc("/api/content", s.handleContent)
	mux.HandleFunc("/api/config", s.handleConfig)

	slog.Info("Starting web server", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	configInfo := map[string]any{
		"llm_provider":       s.cfg.LLM.Provider,
		"llm_model":          s.cfg.LLM.Model,
		"embedding_provider": s.cfg.Embedding.Provider,
		"embedding_model":    s.cfg.Embedding.Model,
		"qdrant_collection":  s.cfg.Qdrant.Collection,
	}

	if info, err := runtime.GetCollectionInfo(s.cfg); err == nil {
		configInfo["collection_status"] = info.Status
		configInfo["collection_points"] = info.PointsCount
		configInfo["collection_vector_size"] = info.VectorSize
		configInfo["collection_segments"] = info.Segments
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configInfo)
}

func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	fullPath, err := vault.ResolvePath(s.cfg.VaultPath, path)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		slog.Error("failed to read file", "path", path, "error", err)
		http.Error(w, "failed to read file", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	topN := s.cfg.Search.MaxResults
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if n, err := strconv.Atoi(nStr); err == nil {
			topN = n
		}
	}

	results, err := searcher.Search(r.Context(), s.store, query, topN, float32(s.cfg.Search.MinScore))
	if err != nil {
		slog.Error("search failed", "query", query, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleSummarize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	summary, err := summarizer.Summarize(
		r.Context(),
		s.store,
		s.llm,
		query,
		s.cfg.Summary.TopK,
		s.cfg.VaultPath,
		s.cfg.LLM.MaxTokens,
	)
	if err != nil {
		slog.Error("summarize failed", "query", query, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"summary": summary})
}
