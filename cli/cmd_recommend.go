package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) recommendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recommend <id>",
		Short: "Show papers recommended as similar to this one",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching recommendations for %q...", args[0])
			papers, err := a.client.Recommend(cmd.Context(), args[0], n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(papers, len(papers))
		},
	}
	return cmd
}
