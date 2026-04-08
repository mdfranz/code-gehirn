package tui

import "github.com/mfranz/code-gehirn/internal/searcher"

// SearchResultMsg is dispatched when an async search completes.
type SearchResultMsg struct {
	Query   string
	Request uint64
	Results []searcher.Result
	Err     error
}

// SummaryMsg is dispatched when the LLM summarization completes.
// Query must match SummaryModel.query or the message is a stale result and
// should be discarded.
type SummaryMsg struct {
	Query string
	Text  string
	Err   error
}

// StatusMsg updates the status bar text in the root AppModel.
type StatusMsg struct {
	Text string
}

// doSearchMsg triggers an actual search after the debounce timer fires.
type doSearchMsg struct{ query string }

// switchToSummaryMsg is sent by SearchModel to the root AppModel
// to trigger a screen transition to the summary view.
type switchToSummaryMsg struct{ query string }
