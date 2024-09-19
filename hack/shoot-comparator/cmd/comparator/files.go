package comparator

import (
	"fmt"

	"github.com/kyma-project/infrastructure-manager/tools/shoot-comparator/internal/files"
	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:     "files",
	Aliases: []string{"f"},
	Short:   "Compare files",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		leftFile := args[0]
		rightFile := args[1]

		fmt.Printf("Comparing files: %s and %s \n", leftFile, rightFile)
		equal, matcherErrorMessage, err := files.CompareFiles(leftFile, rightFile)
		if err != nil {
			fmt.Printf("Failed to compare files: %s", err.Error())
			return
		}

		if equal {
			fmt.Println("Shoot files are equal")
		} else {
			fmt.Println("Shoot files are NOT equal")
			fmt.Println(matcherErrorMessage)
		}
	},
}

func init() {
	rootCmd.AddCommand(filesCmd)
}
