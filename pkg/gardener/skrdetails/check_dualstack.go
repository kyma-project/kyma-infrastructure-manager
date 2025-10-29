package skrdetails

import (
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
)

func IsDualStackEnabled(shoot *gardener.Shoot) bool {
	if shoot.Spec.Networking == nil {
		return false
	}

	if slices.Contains(shoot.Spec.Networking.IPFamilies, gardener.IPFamilyIPv4) &&
		slices.Contains(shoot.Spec.Networking.IPFamilies, gardener.IPFamilyIPv6) && len(shoot.Spec.Networking.IPFamilies) == 2 {
		return true
	}

	return false
}
