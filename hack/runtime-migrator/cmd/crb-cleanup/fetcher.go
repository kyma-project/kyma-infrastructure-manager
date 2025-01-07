package main

import (
	"context"

	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Fetcher interface {
	FetchNew(context.Context) ([]v1.ClusterRoleBinding, error)
	FetchOld(context.Context) ([]v1.ClusterRoleBinding, error)
}

type KubeLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ClusterRoleBindingList, error)
}

type CRBFetcher struct {
	labelNew string
	labelOld string
	client   KubeLister
}

func (f CRBFetcher) fetch(ctx context.Context, label string) ([]v1.ClusterRoleBinding, error) {
	list, err := f.client.List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})

	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func (f CRBFetcher) FetchNew(ctx context.Context) ([]v1.ClusterRoleBinding, error) {
	return f.fetch(ctx, f.labelNew)
}

func (f CRBFetcher) FetchOld(ctx context.Context) ([]v1.ClusterRoleBinding, error) {
	return f.fetch(ctx, f.labelOld)
}

func NewCRBFetcher(client KubeLister, labelOld, labelNew string) Fetcher {
	return CRBFetcher{
		labelNew: labelNew,
		labelOld: labelOld,
		client:   client,
	}
}
