package utils

import (
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"os/exec"
	"path"
)

func pathToMakefile() (string, error) {
	var dir string
	//set the working directory to the folder where the Makefile is located
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}
	dir, err = recursiveFileLookup(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to find KIM's Makefile (lookup started at '%s'): %v", cwd, err)
	}
	return dir, nil
}

func recursiveFileLookup(lookupPath string) (string, error) {
	if _, err := os.Stat(path.Join(lookupPath, "Makefile")); errors.Is(err, os.ErrNotExist) {
		if lookupPath == "/" {
			return "", os.ErrNotExist
		}
		return recursiveFileLookup(path.Dir(lookupPath))
	}
	return lookupPath, nil
}

func CallMake(cmd *exec.Cmd) error {
	//get working directory
	cwd, err := pathToMakefile()
	if err != nil {
		return err
	}
	cmd.Dir = cwd

	//execute the make command
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to execute make - returned value: %v (%s)", err, output)
	}

	return nil
}

func CreateRuntimeFromFile(path string) (*imv1.Runtime, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var runtime imv1.Runtime
	if err := yaml.Unmarshal(data, &runtime); err != nil {
		return nil, err
	}
	return &runtime, nil
}
