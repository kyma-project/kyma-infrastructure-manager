package fsm

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
)

func getWriterForFilesystem(filePath string) (io.Writer, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create file: %w", err)
	}
	return file, nil
}

func persist(path string, s interface{}, saveFunc writerGetter) error {
	writer, err := saveFunc(path)
	if err != nil {
		return fmt.Errorf("unable to create file: %w", err)
	}

	b, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("unable to marshal shoot: %w", err)
	}

	if _, err = writer.Write(b); err != nil {
		return fmt.Errorf("unable to write to file: %w", err)
	}
	return nil
}

func sFnDumpShootSpec(_ context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	paths := createFilesPath(m.PVCPath, s.shoot.Namespace, s.shoot.Name)

	// To make comparison easier we don't store object obtained from the cluster as it contains additional fields that are not relevant for the comparison.
	// We use object created by the converter instead (the Provisioner uses the same approach)
	convertedShoot, err := convertShoot(&s.instance, m.Config.ConverterConfig, s.shoot)
	if err != nil {
		return updateStatusAndStopWithError(err)
	}

	convertedShoot.ObjectMeta.CreationTimestamp = metav1.Time{
		Time: time.Now(),
	}

	runtimeCp := s.instance.DeepCopy()

	if err := persist(paths["shoot"], convertedShoot, m.writerProvider); err != nil {
		return updateStatusAndStopWithError(err)
	}

	if err := persist(paths["runtime"], runtimeCp, m.writerProvider); err != nil {
		return updateStatusAndStopWithError(err)
	}
	return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)
}

func createFilesPath(pvcPath, namespace, name string) map[string]string {
	m := make(map[string]string)
	m["shoot"] = fmt.Sprintf("%s/%s-%s-shootCR.yaml", pvcPath, namespace, name)
	m["runtime"] = fmt.Sprintf("%s/%s-%s-runtimeCR.yaml", pvcPath, namespace, name)
	return m
}
