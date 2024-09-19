package comparator

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/infrastructure-manager/tools/shoot-comparator/internal/directories"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(directoriesCmd)
	directoriesCmd.Flags().String("fromDate", "", "Files older than specified date will not be compared")
	directoriesCmd.Flags().String("outputDir", "", "Directory storing comparison results")
}

var directoriesCmd = &cobra.Command{
	Use:     "dirs",
	Aliases: []string{"d"},
	Short:   "Compares directories",
	Args:    cobra.ExactArgs(2),
	Long: "Compares two directories and checks if they are equal. " +
		"By default, all files are compared. " +
		"Use the --fromDate flag to specify the date from which the comparison should be performed. " +
		"Use the --outputDir flag to save the comparison results.",
	Run: func(cmd *cobra.Command, args []string) {
		leftDir := args[0]
		rightDir := args[1]

		fromDateString, err := cmd.Flags().GetString("fromDate")
		if err != nil {
			fmt.Printf("Error occurred when parsing command line arguments:%q", err)
			return
		}

		fromDate, err := parseFromDate(fromDateString)
		if err != nil {
			slog.Error(fmt.Sprintf("Error occurred when parsing command line arguments: %q", err))
			return
		}

		outputDir, err := cmd.Flags().GetString("outputDir")
		if err != nil {
			slog.Error(fmt.Sprintf("Error occurred when parsing command line arguments: %q", err))
			return
		}

		if !fromDate.IsZero() {
			slog.Info(fmt.Sprintf("Only files created after the following date: %v will be compared.", fromDate))
		}

		slog.Info(fmt.Sprintf("Comparing directories: %s and %s", leftDir, rightDir))
		result, err := directories.CompareDirectories(leftDir, rightDir, time.Time{})
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to compare directories: %q", err.Error()))
			return
		}

		if outputDir != "" {
			writeReport(outputDir, result, fromDate)
		}
		logComparisonResults(result)
	},
}

func writeReport(outputDir string, result directories.Result, fromDate time.Time) {
	slog.Info("Saving comparison details")
	resultsDir, err := directories.SaveComparisonReport(result, outputDir, fromDate)
	if err != nil {
		fmt.Printf("Failed to save comparison report: %s \n", err.Error())
	} else {
		slog.Info(fmt.Sprintf("Results stored in %q", resultsDir))
	}
}

func parseFromDate(fromDateString string) (time.Time, error) {
	if fromDateString != "" {
		fromDate, err := time.Parse(time.RFC3339, fromDateString)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse `fromDate': %q", err)
		}

		return fromDate, nil
	}

	return time.Time{}, nil
}

func logComparisonResults(comparisonResult directories.Result) {
	if comparisonResult.Equal {
		slog.Info("No differences found.")
	} else {
		slog.Warn("Differences found.")
	}
}
