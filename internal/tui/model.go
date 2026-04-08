package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/mfranz/code-gehirn/internal/provider"
	"github.com/mfranz/code-gehirn/internal/store"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

type screen int

const (
	screenSearch screen = iota
	screenSummary
)

// AppModel is the root Bubble Tea model. It owns store/LLM references and
// delegates to sub-models based on the active screen.
type AppModel struct {
	config       config.Config
	embedder     embeddings.Embedder
	store        qdrant.Store
	llm          llms.Model
	width        int
	height       int
	activeScreen screen
	searchModel  SearchModel
	summaryModel SummaryModel
	logs         []string
	initializing bool
}

func NewAppModel(cfg config.Config) AppModel {
	return AppModel{
		config:       cfg,
		activeScreen: screenSearch,
		logs:         []string{},
		initializing: true,
	}
}

type initStageMsg struct {
	stage   string
	payload interface{}
	err     error
	detail  string
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return LogMsg{Message: fmt.Sprintf("Creating %s embedder (%s)...", m.config.Embedding.Provider, m.config.Embedding.Model)}
		},
		m.initEmbedderCmd(),
	)
}

func (m AppModel) initEmbedderCmd() tea.Cmd {
	return func() tea.Msg {
		embedder, err := provider.NewEmbedder(m.config.Embedding)
		return initStageMsg{stage: "embedder", payload: embedder, err: err}
	}
}

func (m AppModel) initLLMCmd() tea.Cmd {
	return func() tea.Msg {
		llm, err := provider.NewLLM(m.config.LLM)
		return initStageMsg{stage: "llm", payload: llm, err: err}
	}
}

func (m AppModel) initStoreCmd() tea.Cmd {
	return func() tea.Msg {
		qdrantStore, err := store.New(m.config.Qdrant, m.embedder)
		return initStageMsg{stage: "store", payload: qdrantStore, err: err}
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initStageMsg:
		if msg.err != nil {
			m.logs = append(m.logs, errorStyle.Render("Error: "+msg.err.Error()))
			return m, tea.Quit
		}

		// Mark previous step as success
		if len(m.logs) > 0 {
			m.logs[len(m.logs)-1] += " success."
		}

		switch msg.stage {
		case "embedder":
			m.embedder = msg.payload.(embeddings.Embedder)
			return m, tea.Batch(
				func() tea.Msg {
					return LogMsg{Message: fmt.Sprintf("Connecting to %s LLM (%s)...", m.config.LLM.Provider, m.config.LLM.Model)}
				},
				m.initLLMCmd(),
			)
		case "llm":
			m.llm = msg.payload.(llms.Model)
			return m, tea.Batch(
				func() tea.Msg {
					return LogMsg{Message: fmt.Sprintf("Connecting to Qdrant (%s)...", m.config.Qdrant.URL)}
				},
				m.initStoreCmd(),
			)
		case "store":
			m.store = msg.payload.(qdrant.Store)
			m.initializing = false
			m.searchModel = newSearchModel(m.store, float32(m.config.Search.MinScore))
			m.summaryModel = newSummaryModel()
			m.searchModel.SetSize(m.width, m.height-1)
			m.logs = append(m.logs, "Connected to brain.")
			return m, m.searchModel.Init()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if !m.initializing {
			m.searchModel.SetSize(msg.Width, msg.Height-1)
			m.summaryModel.SetSize(msg.Width, msg.Height-1)
		}
		return m, nil

	case LogMsg:
		m.logs = append(m.logs, msg.Message)
		if len(m.logs) > 5 {
			m.logs = m.logs[1:]
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.initializing || m.activeScreen == screenSearch {
				return m, tea.Quit
			}
		case "esc":
			if m.activeScreen == screenSummary {
				m.activeScreen = screenSearch
				return m, nil
			}
		}

	case switchToSummaryMsg:
		m.activeScreen = screenSummary
		m.summaryModel = newSummaryModel()
		m.summaryModel.SetSize(m.width, m.height-1)
		m.logs = append(m.logs, fmt.Sprintf("Preparing brain to summarize '%s'...", msg.query))
		return m, m.summaryModel.startSummary(msg.query, m.store, m.llm, m.config.VaultPath, m.config.LLM.MaxTokens)
	}

	if m.initializing {
		return m, nil
	}

	var cmd tea.Cmd
	switch m.activeScreen {
	case screenSearch:
		m.searchModel, cmd = m.searchModel.Update(msg)
	case screenSummary:
		m.summaryModel, cmd = m.summaryModel.Update(msg)
	}
	return m, cmd
}

func (m AppModel) View() string {
	if m.width == 0 {
		return ""
	}

	var view string
	if m.initializing {
		var sb strings.Builder
		sb.WriteString("\n\n  " + titleStyle.Render(" Initializing brain ") + "\n\n")
		for _, log := range m.logs {
			sb.WriteString("  " + dimStyle.Render("•") + " " + log + "\n")
		}
		view = sb.String()
		// Fill remaining space
		lines := strings.Split(view, "\n")
		for i := len(lines); i < m.height-1; i++ {
			view += "\n"
		}
	} else {
		switch m.activeScreen {
		case screenSummary:
			view = m.summaryModel.View()
		default:
			view = m.searchModel.View()
		}
	}

	// Pad view to fill height (minus 1 line for status bar)
	viewHeight := lipgloss.Height(view)
	if viewHeight < m.height-1 {
		view += strings.Repeat("\n", m.height-1-viewHeight)
	}

	// Status bar
	status := "Ready"
	if m.initializing {
		status = "System startup..."
	} else if len(m.logs) > 0 {
		status = m.logs[len(m.logs)-1]
	}
	sb := statusBarStyle.Width(m.width).Render(statusTextStyle.Render(" [brain] " + status))

	return lipgloss.JoinVertical(lipgloss.Left,
		view,
		sb,
	)
}
