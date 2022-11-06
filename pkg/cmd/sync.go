package cmd

import (
	"fmt"
	"os"

	"github.com/marcelbeumer/gonotes/pkg/gonotes"
	"github.com/spf13/cobra"
)

var dryRun bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync repository",
	Long:  "Sync repository writing notes to correct paths, updating tag tree, etc.",
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

		if verbose {
			fmt.Println("Planning operations.")
		}

		plan, err := repo.Plan()
		if err != nil {
			handleError(err)
		}
		if verbose {
			fmt.Println(plan.String())
		}

		if dryRun {
			if verbose {
				fmt.Println("Dry run, exiting.")
			}
			return
		}

		if verbose {
			fmt.Println("Applying.")
		}

		done, err := repo.Apply(plan, os.Stdout)
		if err != nil {
			handleError(err)
		}
		if !done {
			os.Exit(1)
		}

		if verbose {
			fmt.Println("Done.")
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.PersistentFlags().BoolVar(&dryRun, "dry", false, "dry run (plan but do not apply)")
}
