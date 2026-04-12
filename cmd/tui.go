package cmd

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mfranz/code-gehirn/internal/logger"
	tuipkg "github.com/mfranz/code-gehirn/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Redirect stderr to app.log so provider SDK output doesn't corrupt the TUI display.
		if lf := logger.LogFile(); lf != nil {
			origStderr := os.Stderr
			os.Stderr = lf
			defer func() { os.Stderr = origStderr }()
		}
		m := tuipkg.NewAppModel(*cfg)
		p := tea.NewProgram(m, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		if am, ok := finalModel.(tuipkg.AppModel); ok {
			return am.FatalErr()
		}
		return nil
	},
}
