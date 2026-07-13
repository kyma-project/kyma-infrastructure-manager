package gdch

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type InfrastructureConfig struct {
	metav1.TypeMeta `json:",inline"`
	EnableEgress    bool          `json:"enableEgress"`
	Networks        NetworkConfig `json:"networks"`
}

type ControlPlaneConfig struct {
	metav1.TypeMeta `json:",inline"`
}

type NetworkConfig struct {
	NodeCIDR        string          `json:"nodeCIDR"`
	Zones           []Zone          `json:"zones"`
	ParentReference ParentReference `json:"parentReference"`
}

type Zone struct {
	Name string `json:"name"`
	CIDR string `json:"CIDR"`
}

type ParentReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"` // Defaults to the project namespace of the service account
	Type      string `json:"type,omitempty"`      // Defaults to SingleSubnet
}
