package cmd

import (
	"fmt"
	"log"

	"github.com/marcelbeumer/gonotes/internal/gonotes"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(lastCmd)
}

var lastCmd = &cobra.Command{
	Use:   "last",
	Short: "Last note path",
	Long:  "Print path of last note in repository.",
	Run: func(cmd *cobra.Command, args []string) {
		repo := gonotes.NewRepository()
		err := repo.LoadPaths()
		if err != nil {
			log.Fatal(err)
		}
		err = repo.LoadNotes()
		if err != nil {
			handleError(err)
		}
		last, err := repo.LastNote()
		if err != nil {
			handleError(err)
		}
		notePath := repo.NotePaths()[last]
		fmt.Println(notePath)
	},
}
