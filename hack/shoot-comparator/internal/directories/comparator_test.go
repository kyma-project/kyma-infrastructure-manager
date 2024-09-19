package directories

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const onlyLeftFilename = "onlyLeftFile.yaml"
const onlyRightFileName = "onlyRightFileName.yaml"
const equalFileName = "equalFile.yaml"
const differentFileName = "differentFile.yaml"

func TestCompareDirectories(t *testing.T) {

	for _, testCase := range []struct {
		description                string
		dateFromFunc               func() time.Time
		expectedLeftOnly           []string
		expectedRightOnly          []string
		expectedDifferentFileNames []string
		expectedEqual              bool
	}{
		{
			description:                "should return comparison results for two directories",
			dateFromFunc:               func() time.Time { return time.Time{} },
			expectedLeftOnly:           []string{onlyLeftFilename},
			expectedRightOnly:          []string{onlyRightFileName},
			expectedDifferentFileNames: []string{differentFileName},
			expectedEqual:              false,
		},
		{
			description:                "should return empty comparison results when files were created before specific date",
			dateFromFunc:               func() time.Time { return time.Now() },
			expectedLeftOnly:           nil,
			expectedRightOnly:          nil,
			expectedDifferentFileNames: nil,
			expectedEqual:              true,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			leftDir, rightDir := fixTestDataInTempDir(t)

			// when
			result, err := CompareDirectories(leftDir, rightDir, testCase.dateFromFunc())
			require.NoError(t, err, "Failed to compare directories")

			// then
			assert.Equal(t, leftDir, result.LeftDir, "Expected LeftDir to be %s, but got %s", leftDir, result.LeftDir)
			assert.Equal(t, rightDir, result.RightDir, "Expected RightDir to be %s, but got %s", rightDir, result.RightDir)
			assert.Equal(t, testCase.expectedEqual, result.Equal, "Expected Equal to be true, but got false")
			assert.Equal(t, testCase.expectedLeftOnly, result.LeftOnly, "Expected LeftOnly to contain %s, but got %v", testCase.expectedLeftOnly, result.LeftOnly)
			assert.Equal(t, testCase.expectedRightOnly, result.RightOnly, "Expected RightOnly to contain %s, but got %v", testCase.expectedRightOnly, result.RightOnly)
			assert.Equal(t, len(testCase.expectedDifferentFileNames), len(result.Diff), "Expected Diff to contain %d elements, but got %d", len(testCase.expectedDifferentFileNames), len(result.Diff))

			if len(testCase.expectedDifferentFileNames) != 0 {
				for i, diffFileName := range testCase.expectedDifferentFileNames {
					leftFilePath := path.Join(leftDir, diffFileName)
					rightFilePath := path.Join(rightDir, diffFileName)

					assert.Equal(t, testCase.expectedDifferentFileNames[i], result.Diff[i].Filename, "Expected Diff to contain %s, but got %v", testCase.expectedDifferentFileNames[i], result.Diff[i].Filename)
					assert.Equal(t, leftFilePath, result.Diff[i].LeftFile, "Expected Diff to contain %s, but got %v", leftFilePath, result.Diff[i].LeftFile)
					assert.Equal(t, rightFilePath, result.Diff[i].RightFile, "Expected Diff to contain %s, but got %v", rightFilePath, result.Diff[i].RightFile)
					assert.NotEmpty(t, result.Diff[i].Message, "Expected Diff to contain message")
				}
			}
		})
	}

	for _, testCase := range []struct {
		description string
		getDirsFunc func() (string, string)
	}{
		{
			description: "should return error when failed to list files in left directory",
			getDirsFunc: func() (string, string) {
				_, rightDir := fixTestDataInTempDir(t)

				return "non-existing-directory", rightDir
			},
		},
		{
			description: "should return error when failed to list files in right directory",
			getDirsFunc: func() (string, string) {
				leftDir, _ := fixTestDataInTempDir(t)

				return leftDir, "non-existing-directory"
			},
		},
	} {
		t.Run("should return error when failed to list files", func(t *testing.T) {
			// given
			_, rightDir := testCase.getDirsFunc()

			// when
			_, err := CompareDirectories("non-existing-directory", rightDir, time.Time{})

			// then
			require.Error(t, err, "Expected to return error when failed to list files in directory")
		})
	}
}

func fixTestDataInTempDir(t *testing.T) (string, string) {
	leftDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err, "Failed to create test directory")

	rightDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err, "Failed to create test directory")

	emptyShoot := getEmptyTestShoot()
	shootWithNonEmptySpec := getTestShootWithSpec()

	saveTestShootFile(t, emptyShoot, leftDir, onlyLeftFilename)
	saveTestShootFile(t, emptyShoot, rightDir, onlyRightFileName)
	saveTestShootFile(t, emptyShoot, leftDir, equalFileName)
	saveTestShootFile(t, emptyShoot, rightDir, equalFileName)
	saveTestShootFile(t, emptyShoot, leftDir, differentFileName)
	saveTestShootFile(t, shootWithNonEmptySpec, rightDir, differentFileName)

	return leftDir, rightDir
}

func saveTestShootFile(t *testing.T, shoot v1beta1.Shoot, dir, filename string) {
	filePath := path.Join(dir, filename)

	file, err := os.Create(filePath)
	require.NoError(t, err, "Failed to create test file")

	defer func() {
		if file != nil {
			err := file.Close()
			t.Logf("Failed to close file: %v", err)
		}
	}()

	err = yaml.NewEncoder(file).Encode(shoot)
	require.NoError(t, err, "Failed to save test file")
}

func getEmptyTestShoot() v1beta1.Shoot {
	return v1beta1.Shoot{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Shoot",
			APIVersion: "core.gardener.cloud/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-shoot",
		},
	}
}

func getTestShootWithSpec() v1beta1.Shoot {
	return v1beta1.Shoot{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Shoot",
			APIVersion: "core.gardener.cloud/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-shoot",
		},
		Spec: v1beta1.ShootSpec{
			CloudProfileName: "test-cloud-profile",
		},
	}
}
