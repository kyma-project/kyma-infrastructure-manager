package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path"
	"strconv"

	v1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func difference[T any](a, b []T, eql func(*T, *T) bool) ([]T, []T) {
	var missing []T

	// Find elements in `a` that are not in `b`
	for _, itemA := range a {
		found := false
		for i, itemB := range b {
			if eql(&itemA, &itemB) {
				found = true
				b = append(b[:i], b[i+1:]...)
				break
			}
		}
		if !found {
			missing = append(missing, itemA)
		}
	}

	return missing, b
}

// CRBEquals checks if crbA is included in crbB
func CRBEquals(crbA, crbB *v1.ClusterRoleBinding) bool {
	if crbA.RoleRef != crbB.RoleRef {
		return false
	}

	subjectsMap := make(map[v1.Subject]bool)
	for _, subject := range crbA.Subjects {
		subjectsMap[subject] = true
	}

	for _, subject := range crbB.Subjects {
		if _, ok := subjectsMap[subject]; !ok {
			return false
		}
	}
	return true
}

func setupKubectl(kubeconfig string) *kubernetes.Clientset {
	slog.Info("Loading kubeconfig", "path", kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		slog.Error("Error building kubeconfig", "error", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("Error building clientset", "error", err)
		os.Exit(1)
	}

	return clientset
}

func CRBNames(crbs []v1.ClusterRoleBinding) slog.Attr {
	names := make([]any, len(crbs))
	for i := range crbs {
		names[i] = slog.String(strconv.Itoa(i), crbs[i].Name)
	}

	return slog.Group("crbs", names...)
}

type Filer interface {
	Missing(crbs []v1.ClusterRoleBinding) error
	Removed(crbs []v1.ClusterRoleBinding) error
	Failures(failures []Failure) error
}

type nopFiler struct{}

func (n nopFiler) Failures(failures []Failure) error {
	return nil
}

func (n nopFiler) Missing(crbs []v1.ClusterRoleBinding) error {
	return nil
}

func (n nopFiler) Removed(crbs []v1.ClusterRoleBinding) error {
	return nil
}

func NewNopFiler() Filer {
	return nopFiler{}
}

type JSONFiler struct {
	prefix   string
	missing  io.Writer
	removed  io.Writer
	failures io.Writer
}

// Failures implements Filer.
func (j JSONFiler) Failures(failures []Failure) error {
	if failures == nil || len(failures) <= 0 {
		return nil
	}
	return saveLogs(failures, j.prefix+"failures.json")
}

// Missing implements Filer.
func (j JSONFiler) Missing(crbs []v1.ClusterRoleBinding) error {
	if crbs == nil || len(crbs) <= 0 {
		return nil
	}
	return saveLogs(crbs, j.prefix+"missing.json")
}

// Removed implements Filer.
func (j JSONFiler) Removed(crbs []v1.ClusterRoleBinding) error {
	if crbs == nil || len(crbs) <= 0 {
		return nil
	}
	return saveLogs(crbs, j.prefix+"removed.json")
}

func saveLogs[T any](data T, path_ string) error {
	dir := path.Dir(path_)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}
	file, err := os.Create(path_)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(data)
}

func NewJSONFiler(prefix string) Filer {
	return JSONFiler{
		prefix:   prefix,
		missing:  nil,
		removed:  nil,
		failures: nil,
	}
}
