package cmd

import (
	"os"

	"github.com/marcelbeumer/gonotes/pkg/gonotes"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(treeCmd)
}

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Show tree of objects",
	Long:  "Show tree of objects in this notes repository",
	Run: func(cmd *cobra.Command, args []string) {
		repo := gonotes.NewRepository()
		err := repo.LoadPaths()
		if err != nil {
			handleError(err)
		}
		err = repo.LoadNotes()
		if err != nil {
			handleError(err)
		}
		tree := repo.GetTree()
		tree.Print(os.Stdout)
	},
}
