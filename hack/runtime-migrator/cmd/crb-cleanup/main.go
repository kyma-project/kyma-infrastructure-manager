package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// const LabelSelectorOld = "kyma-project.io/deprecation=to-be-removed-soon,reconciler.kyma-project.io/managed-by=provisioner"
const LabelSelectorOld = "reconciler.kyma-project.io/managed-by=infrastructure-manager"
const LabelSelectorNew = "reconciler.kyma-project.io/managed-by=infrastructure-manager"

func main() {
	// ###################
	// ### Parse flags ###
	// ###################

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		kubeconfig = envKubeconfig
	}

	pretend := flag.Bool("pretend", false, "Don't remove CRBs, print what would be removed (you might want to increase log level)")
	verbose := flag.Bool("verbose", false, "Increase the log level to debug (default: info)")
	kubeconfigFlag := flag.String("kubeconfig", "", fmt.Sprintf("Kubeconfig file path, if not set: %s", kubeconfig))
	flag.Parse()
	if kubeconfigFlag != nil && len(*kubeconfigFlag) > 0 {
		kubeconfig = *kubeconfigFlag
	}

	_ = pretend

	// ###################

	if verbose != nil && *verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	clientset := setupKubectl(kubeconfig)
	var remover RBACRemover
	if *pretend {
		remover = RBACMockRemover{}
	} else {
		remover = RBACRemoverImpl{clientset: clientset}
	}
	ctx := context.Background()

	// ###########################
	// ### List CRBs to remove ###
	// ###########################

	crbsOld, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{
		LabelSelector: LabelSelectorOld,
	})
	if err != nil {
		slog.Error("Error listing old CRBs", "error", err)
		os.Exit(1)
	}

	crbsNew, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{
		LabelSelector: LabelSelectorNew,
	})
	if err != nil {
		slog.Error("Error listing new CRBs", "error", err)
		os.Exit(1)
	}

	// ##############################################
	// ### Remove CRBs with migrated counterparts ###
	// ##############################################

	for _, crb := range crbsOld.Items {
		slog.Debug("Comparing CRB", "name", crb.Name)
		if !existsNew(crb, crbsNew) {
			slog.Warn("CRB new counterpart not found", "name", crb.Name)
			continue
		}
		slog.Debug("CRB exists in new form", "name", crb.Name)
		slog.Info("Removing CRB", "name", crb.Name)
		if err := remover.RemoveClusterRoleBinding(ctx, crb); err != nil {
			slog.Error("Error removing CRB", "name", crb.Name, "error", err)
		}
	}

}

func existsNew(old v1.ClusterRoleBinding, new *v1.ClusterRoleBindingList) bool {
crbs:
	for _, crb := range new.Items {
		subjectsMap := make(map[v1.Subject]bool)
		for _, subject := range crb.Subjects {
			subjectsMap[subject] = true
		}

		for _, subject := range old.Subjects {
			if !subjectsMap[subject] {
				slog.Debug("Subject not found", "subject", subject, "old", old.Name, "new", crb.Name)
				continue crbs
			}
		}

		if old.RoleRef == crb.RoleRef {
			return true
		}
		slog.Debug("RoleRef not found", "old", old.Name, "new", crb.Name)
	}

	return false
}

type RBACRemover interface {
	RemoveClusterRoleBinding(ctx context.Context, crb v1.ClusterRoleBinding) error
}

type RBACRemoverImpl struct {
	clientset kubernetes.Interface
}

func (r RBACRemoverImpl) RemoveClusterRoleBinding(ctx context.Context, crb v1.ClusterRoleBinding) error {
	slog.Info("Removing CRB", "name", crb.Name)
	return r.clientset.RbacV1().ClusterRoleBindings().Delete(ctx, crb.Name, metav1.DeleteOptions{})
}

type RBACMockRemover struct{}

func (r RBACMockRemover) RemoveClusterRoleBinding(_ context.Context, crb v1.ClusterRoleBinding) error {
	slog.Debug("Mock client: [remove CRB]", "name", crb.Name)
	return nil
}

func setupKubectl(kubeconfig string) *kubernetes.Clientset {
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
