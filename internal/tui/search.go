package tui

import (
	"context"
	"fmt"
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
	store    qdrant.Store
	input    textinput.Model
	spinner  spinner.Model
	preview  viewport.Model
	renderer *glamour.TermRenderer
	results  []searcher.Result
	cursor   int
	loading  bool
	status   string
	width    int
	height   int
}

func newSearchModel(store qdrant.Store) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Type to search your markdown knowledge base..."
	ti.Focus()
	ti.Width = 60

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = spinnerStyle

	return SearchModel{
		store:   store,
		input:   ti,
		spinner: sp,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Filter out terminal control sequences/OSC responses
		if strings.HasPrefix(msg.String(), "]11;") || strings.HasPrefix(msg.String(), "\033") {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updatePreview()
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
				m.updatePreview()
			}
			return m, nil

		case "enter":
			if len(m.results) > 0 && m.input.Value() != "" {
				return m, func() tea.Msg {
					return switchToSummaryMsg{query: m.input.Value()}
				}
			}
			return m, nil
		}

		var tiCmd tea.Cmd
		m.input, tiCmd = m.input.Update(msg)
		cmds = append(cmds, tiCmd)

		if m.input.Value() != "" {
			m.loading = true
			cmds = append(cmds, debounceCmd(m.input.Value()))
		} else {
			m.results = nil
			m.loading = false
			m.status = ""
			m.preview.SetContent("")
		}

	case doSearchMsg:
		if msg.query == m.input.Value() {
			cmds = append(cmds, m.searchCmd(msg.query), m.spinner.Tick)
			cmds = append(cmds, func() tea.Msg {
				return LogMsg{Message: fmt.Sprintf("Searching for '%s'...", msg.query)}
			})
		}

	case SearchResultMsg:
		m.loading = false
		if msg.Err != nil {
			m.status = msg.Err.Error()
			cmds = append(cmds, func() tea.Msg {
				return LogMsg{Message: "Search failed: " + msg.Err.Error()}
			})
		} else {
			m.results = msg.Results
			m.cursor = 0
			if len(msg.Results) == 0 {
				m.status = "no results"
				cmds = append(cmds, func() tea.Msg {
					return LogMsg{Message: "No results found."}
				})
			} else {
				m.status = fmt.Sprintf("%d found", len(msg.Results))
				cmds = append(cmds, func() tea.Msg {
					return LogMsg{Message: fmt.Sprintf("Found %d results.", len(msg.Results))}
				})
			}
			m.updatePreview()
		}

	case spinner.TickMsg:
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
	if len(m.results) == 0 || m.cursor >= len(m.results) || m.renderer == nil {
		return
	}
	rendered, err := m.renderer.Render(m.results[m.cursor].Doc.PageContent)
	if err != nil {
		rendered = m.results[m.cursor].Doc.PageContent
	}
	m.preview.SetContent(rendered)
	m.preview.GotoTop()
}

func (m *SearchModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	// Reserve: 1 header + 1 search line + 1 blank + 1 status + results list + 1 divider
	listHeight := 12
	previewHeight := h - 7 - listHeight
	if previewHeight < 4 {
		previewHeight = 4
	}
	m.preview = viewport.New(w-2, previewHeight)

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(w-4),
	)
	if err == nil {
		m.renderer = r
		m.updatePreview()
	}
}

func (m SearchModel) View() string {
	if m.width == 0 {
		return ""
	}
	var sb strings.Builder

	// Header
	title := titleStyle.Render(" code-gehirn ")
	help := dimStyle.Render(" [↑↓/jk] navigate  [enter] summarize  [q] quit ")
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
		label := fmt.Sprintf("[%d] %s", i+1, r.Title)
		pathStr := ""
		if r.Path != "" {
			pathStr = " " + dimStyle.Render("("+r.Path+")")
		}

		if i == m.cursor {
			sb.WriteString(selectedItemStyle.Render("  > "+label) + pathStr + "\n")
		} else {
			sb.WriteString(normalItemStyle.Render("    "+label) + pathStr + "\n")
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

func (m SearchModel) searchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := searcher.Search(context.Background(), m.store, query, 20)
		return SearchResultMsg{Results: results, Err: err}
	}
}
