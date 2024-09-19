package files

import (
	"os"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/tools/shoot-comparator/pkg/shoot"
	"sigs.k8s.io/yaml"
)

func CompareFiles(leftFile, rightFile string) (bool, string, error) {
	var leftObject, rightObject v1beta1.Shoot
	err := readYaml(leftFile, &leftObject)
	if err != nil {
		return false, "", err
	}

	err = readYaml(rightFile, &rightObject)
	if err != nil {
		return false, "", err
	}

	matcher := shoot.NewMatcher(leftObject)

	success, err := matcher.Match(rightObject)
	if err != nil {
		return false, "", err
	}

	return success, matcher.FailureMessage(nil), nil
}

func readYaml(path string, shoot *v1beta1.Shoot) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(bytes, shoot)
}
