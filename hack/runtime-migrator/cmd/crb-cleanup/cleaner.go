package main

import (
	"context"
	"log/slog"

	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubeDeleter interface {
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type Compared struct {
	old        []v1.ClusterRoleBinding
	new        []v1.ClusterRoleBinding
	missing    []v1.ClusterRoleBinding
	additional []v1.ClusterRoleBinding
}

type Cleaner interface {
	Clean(context.Context, []v1.ClusterRoleBinding) []Failure
}

type CRBCleaner struct {
	client KubeDeleter
}

type Failure struct {
	CRB v1.ClusterRoleBinding `json:"crb"`
	Err error                 `json:"error"`
}

// Clean deletes CRBs, returning list of deleting errors
func (c CRBCleaner) Clean(ctx context.Context, crbs []v1.ClusterRoleBinding) []Failure {
	failures := make([]Failure, 0)

	for _, crb := range crbs {
		slog.Debug("Removing CRB", "crb", crb.Name)
		err := c.client.Delete(ctx, crb.Name, metav1.DeleteOptions{})
		if err != nil {
			slog.Error("Error removing CRB", "crb", crb.Name)
			failures = append(failures, Failure{
				CRB: crb,
				Err: err,
			})
		}
	}

	return failures
}

// Compare returns missing, additional and original CRBs
func Compare(ctx context.Context, old []v1.ClusterRoleBinding, new []v1.ClusterRoleBinding) Compared {
	missing, additional := difference(old, new, CRBEquals)

	return Compared{
		old:        old,
		new:        new,
		missing:    missing,
		additional: additional,
	}
}

func NewCRBCleaner(client KubeDeleter) Cleaner {
	return CRBCleaner{
		client: client,
	}
}

type DryCleaner struct {
	filer Filer
}

func (p DryCleaner) Clean(_ context.Context, crbs []v1.ClusterRoleBinding) []Failure {
	slog.Debug("Removing CRBs", "crbs", crbs)
	err := p.filer.Removed(crbs)
	if err != nil {
		slog.Error("Error saving removed CRBs", "error", err, "crbs", CRBNames(crbs))
	}
	return nil
}

func NewDryCleaner(filer Filer) Cleaner {
	return DryCleaner{
		filer: filer,
	}
}
