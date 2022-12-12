package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/marcelbeumer/gonotes/internal/gonotes"
	"github.com/spf13/cobra"
)

var from *string
var to *string

var renameTagCmd = &cobra.Command{
	Use:   "rename-tag",
	Short: "Rename tags.",
	Long:  "Rename tags in the repository",
	Run: func(cmd *cobra.Command, args []string) {
		repo := gonotes.NewRepository()
		if err := repo.LoadPaths(); err != nil {
			log.Fatal(err)
		}
		if err := repo.LoadNotes(); err != nil {
			log.Fatal(err)
		}
		count, err := repo.RenameTag(*from, *to)
		if err != nil {
			log.Fatal(err)
		}
		if count == 0 {
			return
		}
		plan, err := repo.Plan()
		if err != nil {
			handleError(err)
		}
		done, err := repo.Apply(plan, os.Stdout)
		if err != nil {
			handleError(err)
		}
		if !done {
			os.Exit(1)
		}
		fmt.Printf("renamed tags in %d notes\n", count)
	},
}

func init() {
	rootCmd.AddCommand(renameTagCmd)
	from = renameTagCmd.Flags().String("from", "", "From")
	to = renameTagCmd.Flags().String("to", "", "To")
}
