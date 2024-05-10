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

package model

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RuntimeSpec defines the desired state of Runtime
type RuntimeSpec struct {
	Name       string                    `json:"name"`
	Purpose    string                    `json:"purpose"`
	Kubernetes RuntimeKubernetes         `json:"kubernetes"`
	Provider   RuntimeProvider           `json:"provider"`
	Networking RuntimeSecurityNetworking `json:"networking"`
	Workers    []gardener.Worker         `json:"workers,omitempty"`
}

// RuntimeStatus defines the observed state of Runtime
type RuntimeStatus struct {
	// State signifies current state of Runtime
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
	State v1.State `json:"state,omitempty"`

	// List of status conditions to indicate the status of a ServiceInstance.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

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

func init() {
	v1.SchemeBuilder.Register(&Runtime{}, &RuntimeList{})
}

type RuntimeAPIServer struct {
	OidcConfig           gardener.OIDCConfig    `json:"oidcConfig"`
	AdditionalOidcConfig *[]gardener.OIDCConfig `json:"additionalOidcConfig"`
}

type RuntimeKubernetes struct {
	Version       *string           `json:"version,omitempty"`
	KubeAPIServer *RuntimeAPIServer `json:"kubeAPIServer,omitempty"`
}

type RuntimeSecurityNetworking struct {
	Filtering      RuntimeSecurityNetworkingFiltering `json:"filiering"`
	Administrators []string                           `json:"administrators"`
}

type RuntimeSecurityNetworkingFiltering struct {
	Ingress RuntimeSecurityNetworkingFilteringIngress `json:"ingress"`
	Egress  RuntimeSecurityNetworkingFilteringEgress  `json:"egress"`
}

type RuntimeSecurityNetworkingFilteringIngress struct {
	Enabled bool `json:"enabled"`
}

type RuntimeSecurityNetworkingFilteringEgress struct {
	Enabled bool `json:"enabled"`
}

type RuntimeProvider struct {
	Type              string `json:"type"`
	Region            string `json:"region"`
	SecretBindingName string `json:"secretBindingName"`
}
