/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CAIstioSpec defines the desired state of CAIstio
type CAIstioSpec struct {
	// Blockchain Identifier
	ChainID string `json:"chainid,omitempty"`
	// Destination Service(internal node) Name
	NodeService string `json:"nodeservice,omitempty"`
	// Destination ServiceEntry(external node) Name
	NodeServiceEntry string `json:"nodeserviceentry,omitempty"`
	// monitoring taget URL
	MonitorURL string `json:"monitorurl,omitempty"`
}

// CAIstioStatus defines the observed state of CAIstio
type CAIstioStatus struct {
	// Current Destination
	Destination         string `json:"destination"`
	ResponseFailedCount int    `json:"responsefailedcount"`
	LatestBlockHeight   int    `json:"latestblockheight"`
	HeightFailedCount   int    `json:"heightfailedcount"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CAIstio is the Schema for the caistios API
type CAIstio struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CAIstioSpec   `json:"spec,omitempty"`
	Status CAIstioStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CAIstioList contains a list of CAIstio
type CAIstioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CAIstio `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CAIstio{}, &CAIstioList{})
}
