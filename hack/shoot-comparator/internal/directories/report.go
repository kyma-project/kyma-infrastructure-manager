package directories

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"
)

func SaveComparisonReport(comparisonResult Result, outputDir string, fromDate time.Time) (string, error) {
	resultsDir, err := createOutputDir(outputDir)
	if err != nil {
		return "", err
	}

	report, err := NewReport(resultsDir)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := report.Close(); err != nil {
			slog.Error(fmt.Sprintf("Failed to close report file: %q", err))
		}
	}()

	writeSummary(&report, comparisonResult, fromDate)

	if !comparisonResult.Equal {
		err := writeResultsToTheReportFile(&report, comparisonResult)
		if err != nil {
			return "", err
		}

		err = writeResultsToDiffFiles(comparisonResult.Diff, resultsDir)
		if err != nil {
			return "", fmt.Errorf("failed to write files with detected differences: %v", err)
		}
	}

	return resultsDir, report.Save()
}

func writeSummary(report *Report, comparisonResult Result, fromDate time.Time) {
	report.AddLine(fmt.Sprintf("Comparing files older than:%v", fromDate))
	report.AddLine("")

	numberOfFilesLeftMsg := fmt.Sprintf("Number of files in %s directory = %d", comparisonResult.LeftDir, comparisonResult.LeftDirFilesCount)
	report.AddLine(numberOfFilesLeftMsg)

	numberOfFilesRightMsg := fmt.Sprintf("Number of files in %s directory = %d", comparisonResult.RightDir, comparisonResult.RightDirFilesCount)

	report.AddLine(numberOfFilesRightMsg)
	report.AddLine("")

	if comparisonResult.Equal {
		report.AddLine("No differences found.")
	} else {
		report.AddLine("Differences found.")
	}
}

func createOutputDir(outputDir string) (string, error) {
	resultsDir := path.Join(outputDir, time.Now().Format(time.RFC3339))

	err := os.MkdirAll(resultsDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create results directory: %v", err)
	}

	return resultsDir, nil
}

func writeResultsToTheReportFile(report *Report, comparisonResult Result) error {
	err := writeMissingFilesToReport(report, comparisonResult.LeftDir, comparisonResult.LeftOnly)
	if err != nil {
		return fmt.Errorf("failed to write results to file: %v", err)
	}

	err = writeMissingFilesToReport(report, comparisonResult.RightDir, comparisonResult.RightOnly)
	if err != nil {
		return fmt.Errorf("failed to write results to file: %v", err)
	}

	err = writeDifferencesToReport(report, comparisonResult.Diff)
	if err != nil {
		return fmt.Errorf("failed to write results to file: %v", err)
	}

	return nil
}

func writeMissingFilesToReport(report *Report, dir string, missingFiles []string) error {
	if len(missingFiles) == 0 {
		return nil
	}
	report.AddLine("")

	report.AddLine("------------------------------------------------------------------------------------------")
	report.AddLine(fmt.Sprintf("Files existing in %s folder only:", dir))

	for _, missingFile := range missingFiles {
		report.AddLine(missingFile)
	}

	report.AddLine("------------------------------------------------------------------------------------------")

	return nil
}

func writeDifferencesToReport(report *Report, differences []Difference) error {
	if len(differences) == 0 {
		return nil
	}

	report.AddLine("")
	report.AddLine("------------------------------------------------------------------------------------------")
	report.AddLine("Files that differ: ")

	for _, difference := range differences {
		msg := fmt.Sprintf("-%s", difference.Filename)

		report.AddLine(msg)
	}

	report.AddLine("------------------------------------------------------------------------------------------")

	return nil
}

func writeResultsToDiffFiles(differences []Difference, resultsDir string) error {
	writeAndCloseFunc := func(filePath string, text string) error {
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer func() {
			if file != nil {
				err := file.Close()
				if err != nil {
					slog.Error(fmt.Sprintf("Failed to close file: %v", err))
				}
			}
		}()

		_, err = file.Write([]byte(text))

		return err
	}

	for _, difference := range differences {
		diffFile := path.Join(resultsDir, fmt.Sprintf("%s.diff", difference.Filename))

		err := writeAndCloseFunc(diffFile, difference.Message)
		if err != nil {
			return err
		}
	}

	return nil
}

type Report struct {
	reportFile *os.File
	contents   string
}

func NewReport(resultsDir string) (Report, error) {
	resultsFile := path.Join(resultsDir, "results.txt")

	file, err := os.Create(resultsFile)
	if err != nil {
		return Report{}, fmt.Errorf("failed to create results file: %v", err)
	}

	return Report{
		reportFile: file,
	}, nil
}

func (rw *Report) AddLine(line string) {
	rw.contents += line + "\n"
}

func (rw *Report) Save() error {
	_, err := rw.reportFile.Write([]byte(rw.contents))

	return err
}

func (rw *Report) Close() error {
	if rw.reportFile != nil {
		return rw.reportFile.Close()
	}

	return nil
}
