/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="STATE",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="SHOOT-NAME",type=string,JSONPath=`.metadata.labels.kyma-project\.io/shoot-name`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Runtime is the Schema for the runtimes API
type Runtime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuntimeSpec   `json:"spec,omitempty"`
	Status RuntimeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RuntimeList contains a list of Runtime
type RuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runtime `json:"items"`
}

// RuntimeSpec defines the desired state of Runtime
type RuntimeSpec struct {
	Shoot    RuntimeShoot `json:"shoot"`
	Security Security     `json:"security"`
}

// RuntimeStatus defines the observed state of Runtime
type RuntimeStatus struct {
	// State signifies current state of Runtime
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
	State State `json:"state,omitempty"`

	// List of status conditions to indicate the status of a ServiceInstance.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type RuntimeShoot struct {
	Name              string                `json:"name"`
	Purpose           gardener.ShootPurpose `json:"purpose"`
	Region            string                `json:"region"`
	LicenceType       *string               `json:"licenceType,omitempty"`
	SecretBindingName string                `json:"secretBindingName"`
	Kubernetes        Kubernetes            `json:"kubernetes"`
	Provider          Provider              `json:"provider"`
	Networking        Networking            `json:"networking"`
	ControlPlane      gardener.ControlPlane `json:"controlPlane"`
}

type Kubernetes struct {
	Version       *string   `json:"version,omitempty"`
	KubeAPIServer APIServer `json:"kubeAPIServer,omitempty"`
}

type APIServer struct {
	OidcConfig           gardener.OIDCConfig    `json:"oidcConfig"`
	AdditionalOidcConfig *[]gardener.OIDCConfig `json:"additionalOidcConfig,omitempty"`
}

type Provider struct {
	Type    string            `json:"type"`
	Zones   []string          `json:"zones"`
	Workers []gardener.Worker `json:"workers"`
}

type Networking struct {
	Pods     string `json:"pods"`
	Nodes    string `json:"nodes"`
	Services string `json:"services"`
}

type Security struct {
	Administrators []string           `json:"administrators"`
	Networking     NetworkingSecurity `json:"networking"`
}

type NetworkingSecurity struct {
	Filter Filter `json:"filter"`
}

type Filter struct {
	Ingress *Ingress `json:"ingress,omitempty"`
	Egress  Egress   `json:"egress"`
}

type Ingress struct {
	Enabled bool `json:"enabled"`
}

type Egress struct {
	Enabled bool `json:"enabled"`
}

func init() {
	SchemeBuilder.Register(&Runtime{}, &RuntimeList{})
}
