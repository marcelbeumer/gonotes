package cmd

import (
	"fmt"
	"log"

	"github.com/marcelbeumer/gonotes/pkg/gonotes"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List objects",
	Long:  "List objects in this notes repository",
	Run: func(cmd *cobra.Command, args []string) {
		repo := gonotes.NewRepository()
		err := repo.LoadPaths()
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range repo.NotePaths() {
			fmt.Println(p)
		}
	},
}
