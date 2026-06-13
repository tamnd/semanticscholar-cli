package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) paperCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "paper <id>",
		Short: "Show a paper's details (accepts S2 paperId, DOI, or arXiv ID)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching paper %q...", args[0])
			paper, err := a.client.Paper(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]any{paper})
		},
	}
}
