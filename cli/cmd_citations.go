package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) citationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "citations <id>",
		Short: "List papers citing this paper",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching citations for %q...", args[0])
			papers, err := a.client.Citations(cmd.Context(), args[0], n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(papers, len(papers))
		},
	}
	return cmd
}
