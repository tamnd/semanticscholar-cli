package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) referencesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "references <id>",
		Short: "List papers referenced by this paper",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching references for %q...", args[0])
			papers, err := a.client.References(cmd.Context(), args[0], n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(papers, len(papers))
		},
	}
	return cmd
}
