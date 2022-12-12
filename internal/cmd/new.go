package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/marcelbeumer/gonotes/internal/gonotes"
	"github.com/spf13/cobra"
)

var tags *[]string
var title *string
var href *string

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "New note",
	Long:  "New note to the repository",
	Run: func(cmd *cobra.Command, args []string) {
		n := gonotes.Note{
			Meta: gonotes.Meta{
				Title: *title,
				Href:  *href,
				Date:  time.Now(),
				Tags:  *tags,
			},
		}
		repo := gonotes.NewRepository()
		if err := repo.LoadPaths(); err != nil {
			log.Fatal(err)
		}
		if err := repo.LoadNotes(); err != nil {
			log.Fatal(err)
		}
		notePath, err := repo.AddNote(n)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(notePath)
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	title = newCmd.Flags().StringP("title", "t", "", "Title")
	tags = newCmd.Flags().StringArrayP("tag", "T", []string{}, "Tags")
	href = newCmd.Flags().String("href", "", "Href")
}
