package main

import (
	"log/slog"
	"os"
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
