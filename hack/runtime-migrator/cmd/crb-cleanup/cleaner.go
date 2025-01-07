package main

import (
	"context"

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
	Compare(ctx context.Context, old []v1.ClusterRoleBinding, new []v1.ClusterRoleBinding) Compared
	Clean(context.Context, []v1.ClusterRoleBinding) []Failure
}

type CRBCleaner struct {
	client KubeDeleter
}

type Failure struct {
	CRB v1.ClusterRoleBinding `json:"crb"`
	Err error                 `json:"error"`
}

func (c CRBCleaner) Clean(ctx context.Context, crbs []v1.ClusterRoleBinding) []Failure {
	failures := make([]Failure, 0)

	for _, crb := range crbs {
		err := c.client.Delete(ctx, crb.Name, metav1.DeleteOptions{})
		if err != nil {
			failures = append(failures, Failure{
				CRB: crb,
				Err: err,
			})
		}
	}

	return failures
}

func (c CRBCleaner) Compare(ctx context.Context, old []v1.ClusterRoleBinding, new []v1.ClusterRoleBinding) Compared {
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
