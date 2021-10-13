/*
Copyright 2021.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=caddyfile

// CaddyfileTemplate is the Schema for the caddyfiletemplates API
type CaddyfileTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data string `json:"data,omitempty"`
}

//+kubebuilder:object:root=true

// CaddyfileTemplateList contains a list of CaddyfileTemplate
type CaddyfileTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CaddyfileTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CaddyfileTemplate{}, &CaddyfileTemplateList{})
}
