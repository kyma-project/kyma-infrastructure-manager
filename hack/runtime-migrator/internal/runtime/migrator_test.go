package runtime

import (
	"github.com/stretchr/testify/require"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestFilterOnlySupportedTypesOfCRBs(t *testing.T) {
	tests := []struct {
		name     string
		input    []rbacv1.ClusterRoleBinding
		expected []rbacv1.ClusterRoleBinding
	}{
		{
			name: "filter out non-cluster-admin roles",
			input: []rbacv1.ClusterRoleBinding{
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user1"},
					},
				},
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "Role",
						Name: "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user1"},
					},
				},
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "view",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user2"},
					},
				},
			},
			expected: []rbacv1.ClusterRoleBinding{
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user1"},
					},
				},
			},
		},
		{
			name: "filter out roles without user subjects",
			input: []rbacv1.ClusterRoleBinding{
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "group1"},
					},
				},
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "sa1"},
					},
				},
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user1"},
					},
				},
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "sa1"},
						{Kind: rbacv1.GroupKind, Name: "group1"},
						{Kind: rbacv1.UserKind, Name: "user1"},
					},
				},
			},
			expected: []rbacv1.ClusterRoleBinding{
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user1"},
					},
				},
				{
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "sa1"},
						{Kind: rbacv1.GroupKind, Name: "group1"},
						{Kind: rbacv1.UserKind, Name: "user1"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterOnlySupportedTypesOfCRBs(tt.input)

			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAdministratorsList(t *testing.T) {
	tests := []struct {
		name     string
		input    []rbacv1.ClusterRoleBinding
		expected []string
	}{
		{
			name: "single user subject",
			input: []rbacv1.ClusterRoleBinding{
				{
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user1"},
					},
				},
			},
			expected: []string{"user1"},
		},
		{
			name: "multiple user subjects",
			input: []rbacv1.ClusterRoleBinding{
				{
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user1"},
						{Kind: rbacv1.UserKind, Name: "user2"},
					},
				},
			},
			expected: []string{"user1", "user2"},
		},
		{
			name: "no user subject",
			input: []rbacv1.ClusterRoleBinding{
				{
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "group1"},
						{Kind: rbacv1.ServiceAccountKind, Name: "sa1"},
					},
				},
			},
			expected: []string{},
		},
		{
			name: "mixed subjects",
			input: []rbacv1.ClusterRoleBinding{
				{
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user1"},
						{Kind: rbacv1.GroupKind, Name: "group1"},
					},
				},
			},
			expected: []string{"user1"},
		},
		{
			name: "duplicate user subjects",
			input: []rbacv1.ClusterRoleBinding{
				{
					Subjects: []rbacv1.Subject{
						{Kind: rbacv1.UserKind, Name: "user1"},
						{Kind: rbacv1.UserKind, Name: "user1"},
					},
				},
			},
			expected: []string{"user1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAdministratorsList(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
