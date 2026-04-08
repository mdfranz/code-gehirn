package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	tuipkg "github.com/mfranz/code-gehirn/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		m := tuipkg.NewAppModel(*cfg)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}
