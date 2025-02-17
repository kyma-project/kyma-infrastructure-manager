package extensions

import (
	"encoding/json"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

const NetworkFilterType = "shoot-networking-filter"

func NewNetworkFilterExtension(egress imv1.Egress) (*gardener.Extension, error) {

	filterProviderConfig := EgressFilterProviderConfig{
		BlackholingEnabled: egress.Enabled,
		Workers:            egress.Workers,
	}

	extensionJSON, err := json.Marshal(filterProviderConfig)
	if err != nil {
		return nil, err
	}

	disabled := !egress.Enabled
	return &gardener.Extension{
		Type: NetworkFilterType,
		Disabled: ptr.To(disabled),
		ProviderConfig: &apimachineryruntime.RawExtension{
			Raw: extensionJSON,
		},
	}, nil

}

// copied from https://github.com/gardener/gardener-extension-shoot-networking-filter/blob/master/pkg/apis/config/types.go#L31
type EgressFilterProviderConfig struct {
	// BlackholingEnabled is a flag to set blackholing or firewall approach.
	BlackholingEnabled bool

	// Workers contains worker-specific block modes
	Workers *imv1.Workers

	// SleepDuration is the time interval between policy updates.
	SleepDuration *metav1.Duration

	// FilterListProviderType specifies how the filter list is retrieved.
	// Supported types are `static` and `download`.
	// TODO: omited for now, confirm if it's fine
	// FilterListProviderType FilterListProviderType

	// StaticFilterList contains the static filter list.
	// Only used for provider type `static`.
	// TODO: omited for now, confirm if it's fine
	// StaticFilterList []Filter

	// DownloaderConfig contains the configuration for the filter list downloader.
	// Only used for provider type `download`.
	// DownloaderConfig *DownloaderConfig

	// EnsureConnectivity configures the removal of seed and/or shoot load balancers IPs from the filter list.
	// TODO: omited for now, confirm if it's fine
	// EnsureConnectivity *EnsureConnectivity
}
