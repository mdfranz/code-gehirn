package tui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/mfranz/code-gehirn/internal/searcher"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

type SearchModel struct {
	store        qdrant.Store
	minScore     float32
	maxResults   int
	input        textinput.Model
	spinner      spinner.Model
	preview      viewport.Model
	renderer     *glamour.TermRenderer
	results      []searcher.Result
	cursor       int
	previewFocus bool
	loading      bool
	status       string
	width        int
	height       int
	cancelSearch context.CancelFunc
	reqSeq       uint64
	activeReq    uint64
}

func newSearchModel(store qdrant.Store, minScore float32, maxResults int) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Type to search your markdown knowledge base..."
	ti.Focus()
	ti.Width = 60

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = spinnerStyle

	return SearchModel{
		store:      store,
		minScore:   minScore,
		maxResults: maxResults,
		input:      ti,
		spinner:    sp,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()

		// 1. Handle navigation first
		switch s {
		case "up":
			if m.cursor > 0 {
				m.cursor--
				m.updatePreview()
			}
			return m, nil

		case "down":
			if m.cursor < len(m.results)-1 {
				m.cursor++
				m.updatePreview()
			}
			return m, nil

		case "pgup":
			m.preview.HalfViewUp()
			return m, nil

		case "pgdn":
			m.preview.HalfViewDown()
			return m, nil

		case "enter":
			if len(m.results) > 0 && m.input.Value() != "" {
				return m, func() tea.Msg {
					return switchToSummaryMsg{query: m.input.Value()}
				}
			}
			return m, nil
		}

		// 2. Filter input to alphanumeric + space only.
		// This naturally breaks OSC sequences which contain non-alphanumeric
		// chars like ;, :, \, and ].
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ') {
					return m, nil
				}
			}
		} else if strings.HasPrefix(s, "\033") || strings.Contains(s, ";") {
			// Extra safety for terminal sequences
			return m, nil
		}

		// 3. Pass through to text input (handles backspace, left/right, etc.)
		var tiCmd tea.Cmd
		m.input, tiCmd = m.input.Update(msg)
		cmds = append(cmds, tiCmd)

		if m.input.Value() != "" {
			m.cancelInFlight()
			m.loading = true
			cmds = append(cmds, debounceCmd(m.input.Value()))
		} else {
			m.cancelInFlight()
			m.results = nil
			m.loading = false
			m.status = ""
			m.preview.SetContent("")
		}

	case doSearchMsg:
		if msg.query == m.input.Value() {
			m.cancelInFlight()
			ctx, cancel := context.WithCancel(context.Background())
			m.cancelSearch = cancel
			m.reqSeq++
			m.activeReq = m.reqSeq
			slog.Info("search", "query", msg.query)
			cmds = append(cmds, m.searchCmd(ctx, msg.query, m.activeReq), m.spinner.Tick)
			cmds = append(cmds, func() tea.Msg { return StatusMsg{Text: "Searching..."} })
		}

	case SearchResultMsg:
		if msg.Request != m.activeReq || msg.Query != m.input.Value() {
			return m, nil
		}
		m.cancelSearch = nil
		m.loading = false
		if msg.Err != nil {
			// Ignore expected cancellation errors from superseded searches.
			if errors.Is(msg.Err, context.Canceled) {
				return m, nil
			}
			slog.Error("search failed", "query", msg.Query, "error", msg.Err)
			m.status = msg.Err.Error()
			cmds = append(cmds, func() tea.Msg { return StatusMsg{Text: "Search error"} })
		} else {
			m.results = msg.Results
			m.cursor = 0
			if len(msg.Results) == 0 {
				m.status = "no results"
				slog.Info("search complete", "query", msg.Query, "results", 0)
				cmds = append(cmds, func() tea.Msg { return StatusMsg{Text: "No results"} })
			} else {
				m.status = fmt.Sprintf("%d found", len(msg.Results))
				slog.Info("search complete", "query", msg.Query, "results", len(msg.Results))
				cmds = append(cmds, func() tea.Msg {
					return StatusMsg{Text: fmt.Sprintf("Found %d results", len(msg.Results))}
				})
			}
			m.updatePreview()
		}

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(msg)
		cmds = append(cmds, spCmd)

	default:
		var vpCmd tea.Cmd
		m.preview, vpCmd = m.preview.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *SearchModel) updatePreview() {
	if len(m.results) == 0 || m.cursor >= len(m.results) {
		return
	}

	content := m.results[m.cursor].Doc.PageContent
	if m.renderer != nil {
		if rendered, err := m.renderer.Render(content); err == nil {
			content = rendered
		}
	}

	m.preview.SetContent(content)
	m.preview.GotoTop()
}

func (m *SearchModel) SetSize(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	m.width = w
	m.height = h

	// Reserve: 1 header + 1 search line + 1 blank + 1 status + results list + 1 divider
	listHeight := 12
	previewHeight := h - 7 - listHeight
	if previewHeight < 4 {
		previewHeight = 4
	}
	m.preview = viewport.New(w-2, previewHeight)

	// Update renderer only if width changed
	wrap := w - 4
	if wrap < 20 {
		wrap = 20
	}

	if m.renderer == nil || wrap != m.width-4 {
		// Use a standard style instead of WithAutoStyle() which can be slow as it
		// probes the terminal for background color.
		r, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(wrap),
		)
		if err == nil {
			m.renderer = r
		}
	}
	m.updatePreview()
}

func (m SearchModel) View() string {
	if m.width == 0 {
		return ""
	}
	var sb strings.Builder

	// Header
	title := titleStyle.Render(" code-gehirn ")
	help := dimStyle.Render(" [↑↓] navigate  [pgup/pgdn] scroll preview  [enter] summarize  [q] quit ")
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(help)
	if gap < 0 {
		gap = 0
	}
	sb.WriteString(title + strings.Repeat(" ", gap) + help + "\n")

	// Search input
	searchLine := "  Search: " + m.input.View()
	if m.loading {
		searchLine += "  " + m.spinner.View()
	}
	sb.WriteString(searchLine + "\n\n")

	// Results list
	if m.status != "" {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  Results (%s)", m.status)) + "\n")
	} else {
		sb.WriteString(dimStyle.Render("  Results") + "\n")
	}

	maxVisible := 10
	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	if start > 0 {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more above", start)) + "\n")
	}

	for i := start; i < len(m.results) && i < start+maxVisible; i++ {
		r := m.results[i]
		score := dimStyle.Render(fmt.Sprintf("%.2f", r.Score))
		label := fmt.Sprintf("[%d] %s", i+1, r.Title)
		pathStr := ""
		if r.Path != "" {
			pathStr = " " + dimStyle.Render("("+r.Path+")")
		}

		if i == m.cursor {
			sb.WriteString(selectedItemStyle.Render("  > "+label) + " " + score + pathStr + "\n")
		} else {
			sb.WriteString(normalItemStyle.Render("    "+label) + " " + score + pathStr + "\n")
		}
	}

	if start+maxVisible < len(m.results) {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more below", len(m.results)-(start+maxVisible))) + "\n")
	}

	// Preview pane
	if len(m.results) > 0 {
		divider := dividerStyle.Render("  " + strings.Repeat("─", m.width-4) + "\n")
		sb.WriteString(divider)
		sb.WriteString(m.preview.View())
	}

	return sb.String()
}

func debounceCmd(query string) tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(_ time.Time) tea.Msg {
		return doSearchMsg{query: query}
	})
}

func (m *SearchModel) cancelInFlight() {
	if m.cancelSearch != nil {
		m.cancelSearch()
		m.cancelSearch = nil
	}
}

func (m SearchModel) searchCmd(ctx context.Context, query string, request uint64) tea.Cmd {
	return func() tea.Msg {
		results, err := searcher.Search(ctx, m.store, query, m.maxResults, m.minScore)
		return SearchResultMsg{Query: query, Request: request, Results: results, Err: err}
	}
}
