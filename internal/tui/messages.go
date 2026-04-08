package tui

import "github.com/mfranz/code-gehirn/internal/searcher"

// SearchResultMsg is dispatched when an async search completes.
type SearchResultMsg struct {
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

// LogMsg is dispatched to update the status bar or log section.
type LogMsg struct {
	Message string
}

// doSearchMsg triggers an actual search after the debounce timer fires.
type doSearchMsg struct{ query string }

// switchToSummaryMsg is sent by SearchModel to the root AppModel
// to trigger a screen transition to the summary view.
type switchToSummaryMsg struct{ query string }
