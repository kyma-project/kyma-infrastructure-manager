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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GardenerCluster is the Schema for the clusters API
type GardenerCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GardenerClusterSpec   `json:"spec"`
	Status GardenerClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GardenerClusterList contains a list of GardenerCluster
type GardenerClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GardenerCluster `json:"items"`
}

// GardenerClusterSpec defines the desired state of GardenerCluster
type GardenerClusterSpec struct {
	Kubeconfig Kubeconfig `json:"kubeconfig"`
	Shoot      Shoot      `json:"shoot"`
}

// Shoot defines the name of the Shoot resource
type Shoot struct {
	Name string `json:"name"`
}

// Kubeconfig defines the desired kubeconfig location
type Kubeconfig struct {
	Secret Secret `json:"secret"`
}

// SecretKeyRef defines the location, and structure of the secret containing kubeconfig
type Secret struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

type State string

const (
	ReadyState      State = "Ready"
	ProcessingState State = "Processing"
	ErrorState      State = "Error"
	DeletingState   State = "Deleting"
)

type ConditionReason string

const (
	ConditionReasonSecretCreated             State = "SecretCreated"
	ConditionReasonSecretNotFound            State = "SecretNotFound"
	ConditionReasonGardenClusterNotRetrieved State = "GardenClusterNotRetrieved"
	ConditionReasonGardenClusterNotFound     State = "GardenClusterNotFound"
)

type ConditionType string

const (
	ConditionTypeUnknown State = "Unknown"
)

// GardenerClusterStatus defines the observed state of GardenerCluster
type GardenerClusterStatus struct {
	// State signifies current state of Gardener Cluster.
	// Value can be one of ("Ready", "Processing", "Error", "Deleting").
	State State `json:"state,omitempty"`

	// List of status conditions to indicate the status of a ServiceInstance.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (cluster *GardenerCluster) UpdateState(r State, s State, msg string) {
	cluster.Status.State = s
	condition := metav1.Condition{
		Type:               string(ConditionTypeUnknown),
		Status:             "True",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            msg,
	}
	meta.SetStatusCondition(&cluster.Status.Conditions, condition)
}

func init() {
	SchemeBuilder.Register(&GardenerCluster{}, &GardenerClusterList{})
}
