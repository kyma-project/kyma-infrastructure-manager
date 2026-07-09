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
	NodeCIDR string  `json:"nodeCIDR"`
	Zones    []Zones `json:"zones"`
}

type Zones struct {
	Name string `json:"name"`
	CIDR string `json:"CIDR"`
}
