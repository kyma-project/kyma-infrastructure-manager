package skrdetails

import (
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/require"
)

func TestAppliedACL(t *testing.T) {
	t.Run("Should return empty slice (nil) when ACL object is nil", func(t *testing.T) {
		// given
		runtime := imv1.Runtime{}

		// when
		got := AppliedACL(runtime)

		// then
		require.Nil(t, got)
	})

	t.Run("Should return empty slice when AllowedCIDRs is empty", func(t *testing.T) {
		// given
		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Kubernetes: imv1.Kubernetes{
						KubeAPIServer: imv1.APIServer{
							ACL: &imv1.ACL{
								AllowedCIDRs: []string{},
							},
						},
					},
				},
			},
		}

		// when
		got := AppliedACL(runtime)

		// then
		require.Equal(t, []string{}, got)
	})

	t.Run("Should return CIDRs when ACL is provided", func(t *testing.T) {
		// given
		expectedCIDRs := []string{"1.2.3.4/32", "5.6.0.0/16"}
		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Kubernetes: imv1.Kubernetes{
						KubeAPIServer: imv1.APIServer{
							ACL: &imv1.ACL{
								AllowedCIDRs: expectedCIDRs,
							},
						},
					},
				},
			},
		}

		// when
		got := AppliedACL(runtime)

		// then
		require.Equal(t, expectedCIDRs, got)
	})

	t.Run("Should return single CIDR when only one is provided", func(t *testing.T) {
		// given
		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Kubernetes: imv1.Kubernetes{
						KubeAPIServer: imv1.APIServer{
							ACL: &imv1.ACL{
								AllowedCIDRs: []string{"10.0.0.0/8"},
							},
						},
					},
				},
			},
		}

		// when
		got := AppliedACL(runtime)

		// then
		require.Equal(t, []string{"10.0.0.0/8"}, got)
	})
}
