package main

import (
	"context"

	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Fetcher interface {
	FetchKim(context.Context) ([]v1.ClusterRoleBinding, error)
	FetchProvisioner(context.Context) ([]v1.ClusterRoleBinding, error)
}

type KubeLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ClusterRoleBindingList, error)
}

type CRBFetcher struct {
	labelKim         string
	labelProvisioner string
	client           KubeLister
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

func (f CRBFetcher) FetchKim(ctx context.Context) ([]v1.ClusterRoleBinding, error) {
	return f.fetch(ctx, f.labelKim)
}

func (f CRBFetcher) FetchProvisioner(ctx context.Context) ([]v1.ClusterRoleBinding, error) {
	return f.fetch(ctx, f.labelProvisioner)
}

func NewCRBFetcher(client KubeLister, labelProvisioner, labelKim string) Fetcher {
	return CRBFetcher{
		labelKim:         labelKim,
		labelProvisioner: labelProvisioner,
		client:           client,
	}
}
