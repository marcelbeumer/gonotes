package cmd

import (
	"fmt"
	"log"

	"github.com/marcelbeumer/gonotes/pkg/gonotes"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(showCmd)
}

var showCmd = &cobra.Command{
	Use:   "show [note ref]",
	Short: "Show object details",
	Long:  "Show object details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := gonotes.NewRepository()
		if err := repo.LoadPaths(); err != nil {
			log.Fatal(err)
		}
		if err := repo.LoadNotes(); err != nil {
			log.Fatal(err)
		}
		note := repo.FindNote(args[0])
		if note == nil {
			handleError(fmt.Errorf("could not find note \"%s\"", args[0]))
		}
		md, err := note.Marhsal()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(md))
	},
}
