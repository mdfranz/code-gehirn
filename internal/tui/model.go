package tui

import (
	"fmt"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/mfranz/code-gehirn/internal/runtime"
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
	// initLogs holds the three initialization stage lines shown during startup.
	// [0]=embedder, [1]=llm, [2]=store
	initLogs     [3]string
	llmReady     bool
	storeReady   bool
	initializing bool
	status       string
}

func NewAppModel(cfg config.Config) AppModel {
	return AppModel{
		config:       cfg,
		activeScreen: screenSearch,
		initializing: true,
		status:       "Initializing...",
		initLogs: [3]string{
			fmt.Sprintf("Creating %s embedder (%s)...", cfg.Embedding.Provider, cfg.Embedding.Model),
			fmt.Sprintf("Connecting to %s LLM (%s)...", cfg.LLM.Provider, cfg.LLM.Model),
			fmt.Sprintf("Connecting to Qdrant (%s)...", cfg.Qdrant.URL),
		},
	}
}

type initStageMsg struct {
	stage   string
	payload interface{}
	err     error
}

func (m AppModel) Init() tea.Cmd {
	// Start embedder and LLM in parallel — they are independent.
	// Store init starts after embedder completes (needs the embedder instance).
	return tea.Batch(m.initEmbedderCmd(), m.initLLMCmd())
}

func (m AppModel) initEmbedderCmd() tea.Cmd {
	return func() tea.Msg {
		embedder, err := runtime.NewEmbedder(m.config)
		return initStageMsg{stage: "embedder", payload: embedder, err: err}
	}
}

func (m AppModel) initLLMCmd() tea.Cmd {
	return func() tea.Msg {
		llm, err := runtime.NewLLM(m.config)
		return initStageMsg{stage: "llm", payload: llm, err: err}
	}
}

func (m AppModel) initStoreCmd() tea.Cmd {
	return func() tea.Msg {
		qdrantStore, err := runtime.NewStore(m.config, m.embedder)
		return initStageMsg{stage: "store", payload: qdrantStore, err: err}
	}
}

// completeInit transitions from initializing to ready once both LLM and store are done.
func (m AppModel) completeInit() (AppModel, tea.Cmd) {
	m.initializing = false
	m.status = "Ready"
	m.searchModel = newSearchModel(m.store, float32(m.config.Search.MinScore), m.config.Search.MaxResults)
	m.summaryModel = newSummaryModel()
	m.searchModel.SetSize(m.width, m.height-1)
	slog.Info("initialization complete")
	return m, m.searchModel.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initStageMsg:
		if msg.err != nil {
			slog.Error("initialization failed", "stage", msg.stage, "error", msg.err)
			switch msg.stage {
			case "embedder":
				m.initLogs[0] = errorStyle.Render(m.initLogs[0] + " FAILED: " + msg.err.Error())
			case "llm":
				m.initLogs[1] = errorStyle.Render(m.initLogs[1] + " FAILED: " + msg.err.Error())
			case "store":
				m.initLogs[2] = errorStyle.Render(m.initLogs[2] + " FAILED: " + msg.err.Error())
			}
			return m, tea.Quit
		}

		switch msg.stage {
		case "embedder":
			m.embedder = msg.payload.(embeddings.Embedder)
			m.initLogs[0] += " done."
			slog.Info("embedder ready", "provider", m.config.Embedding.Provider, "model", m.config.Embedding.Model)
			// Store can start as soon as embedder is ready; LLM may still be in flight.
			return m, m.initStoreCmd()

		case "llm":
			m.llm = msg.payload.(llms.Model)
			m.llmReady = true
			m.initLogs[1] += " done."
			slog.Info("llm ready", "provider", m.config.LLM.Provider, "model", m.config.LLM.Model)
			if m.storeReady {
				return m.completeInit()
			}
			return m, nil

		case "store":
			m.store = msg.payload.(qdrant.Store)
			m.storeReady = true
			m.initLogs[2] += " done."
			slog.Info("store ready", "url", m.config.Qdrant.URL)
			if m.llmReady {
				return m.completeInit()
			}
			return m, nil
		}
		return m, nil

	case StatusMsg:
		m.status = msg.Text
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if !m.initializing {
			m.searchModel.SetSize(msg.Width, msg.Height-1)
			m.summaryModel.SetSize(msg.Width, msg.Height-1)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.activeScreen == screenSummary {
				m.summaryModel.cancelInFlight()
			}
			return m, tea.Quit
		case "q":
			if m.initializing || m.activeScreen == screenSearch {
				return m, tea.Quit
			}
		case "esc":
			if m.activeScreen == screenSummary {
				m.summaryModel.cancelInFlight()
				m.activeScreen = screenSearch
				m.status = "Ready"
				return m, nil
			}
		}

	case switchToSummaryMsg:
		m.summaryModel.cancelInFlight()
		m.activeScreen = screenSummary
		m.summaryModel = newSummaryModel()
		m.summaryModel.SetSize(m.width, m.height-1)
		m.status = fmt.Sprintf("Summarizing '%s'...", msg.query)
		slog.Info("summary started", "query", msg.query)
		return m, m.summaryModel.startSummary(msg.query, m.store, m.llm, m.config.Summary.TopK, m.config.VaultPath, m.config.LLM.MaxTokens)
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
		for _, log := range m.initLogs {
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

	sb := statusBarStyle.Width(m.width).Render(statusTextStyle.Render(" [brain] " + m.status))

	return lipgloss.JoinVertical(lipgloss.Left, view, sb)
}
