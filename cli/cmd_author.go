package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/semanticscholar-cli/semanticscholar"
)

func (a *App) authorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "author <query>",
		Short: "Search authors on Semantic Scholar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("searching authors for %q...", args[0])
			authors, err := a.client.SearchAuthors(cmd.Context(), args[0], n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(authors, len(authors))
		},
	}
	return cmd
}

func (a *App) profileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile <id>",
		Short: "Show an author's profile and recent papers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching author profile %q...", args[0])
			author, papers, err := a.client.AuthorProfile(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			if err := a.render([]semanticscholar.Author{author}); err != nil {
				return err
			}
			if len(papers) > 0 {
				return a.render(papers)
			}
			return nil
		},
	}
}
