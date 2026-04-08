package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/mfranz/code-gehirn/internal/summarizer"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

type SummaryModel struct {
	query   string
	vp      viewport.Model
	spinner spinner.Model
	loading bool
	err     error
	width   int
	height  int
}

func newSummaryModel() SummaryModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle
	return SummaryModel{spinner: sp, loading: true}
}

func (m SummaryModel) startSummary(query string, store qdrant.Store, llm llms.Model) tea.Cmd {
	m.query = query
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			return LogMsg{Message: fmt.Sprintf("Summarizing: %s", query)}
		},
		func() tea.Msg {
			text, err := summarizer.Summarize(context.Background(), store, llm, query, 5)
			return SummaryMsg{Text: text, Err: err}
		},
	)
}

func (m SummaryModel) Update(msg tea.Msg) (SummaryModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case SummaryMsg:
		m.loading = false
		m.err = msg.Err
		if msg.Err == nil {
			cmds = append(cmds, func() tea.Msg {
				return LogMsg{Message: "Summary complete."}
			})
			r, err := glamour.NewTermRenderer(
				glamour.WithStandardStyle("dark"),
				glamour.WithWordWrap(m.width-4),
			)
			content := msg.Text
			if err == nil {
				if rendered, rerr := r.Render(msg.Text); rerr == nil {
					content = rendered
				}
			}
			m.vp.SetContent(content)
			m.vp.GotoTop()
		} else {
			cmds = append(cmds, func() tea.Msg {
				return LogMsg{Message: "Summary failed: " + msg.Err.Error()}
			})
		}
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
}

func (m *SummaryModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.vp = viewport.New(w-2, h-5)
}

func (m SummaryModel) View() string {
	if m.width == 0 {
		return ""
	}
	var sb strings.Builder

	// Header
	title := titleStyle.Render(" code-gehirn ")
	help := dimStyle.Render(" [esc] back  [↑↓/pgup/pgdn] scroll ")
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(help)
	if gap < 0 {
		gap = 0
	}
	sb.WriteString(title + strings.Repeat(" ", gap) + help + "\n\n")

	sb.WriteString(summaryTitleStyle.Render(fmt.Sprintf("  Summary: %s\n", m.query)))
	sb.WriteString(dividerStyle.Render("  " + strings.Repeat("─", m.width-4) + "\n"))

	if m.loading {
		sb.WriteString("  " + m.spinner.View() + " Generating summary...\n")
	} else if m.err != nil {
		sb.WriteString(errorStyle.Render("  Error: "+m.err.Error()) + "\n")
	} else {
		sb.WriteString(m.vp.View())
	}

	return sb.String()
}
