// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package calico

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Group               = "crd.projectcalico.org"
	VersionCurrent      = "v1"
	GroupVersionCurrent = Group + "/" + VersionCurrent

	KindBlockAffinity     = "BlockAffinity"
	KindBlockAffinityList = "BlockAffinityList"
)

// BlockAffinity maintains a block affinity's state
type BlockAffinity struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the BlockAffinity.
	Spec BlockAffinitySpec `json:"spec,omitempty"`
}

// BlockAffinitySpec contains the specification for a BlockAffinity resource.
type BlockAffinitySpec struct {
	State string `json:"state"`
	Node  string `json:"node"`
	CIDR  string `json:"cidr"`

	// Deleted indicates that this block affinity is being deleted.
	// This field is a string for compatibility with older releases that
	// mistakenly treat this field as a string.
	Deleted string `json:"deleted"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlockAffinityList contains a list of BlockAffinity resources.
type BlockAffinityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []BlockAffinity `json:"items"`
}

// NewBlockAffinity creates a new (zeroed) BlockAffinity struct with the TypeMetadata initialised to the current
// version.
func NewBlockAffinity() *BlockAffinity {
	return &BlockAffinity{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindBlockAffinity,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewBlockAffinityList creates a new (zeroed) BlockAffinityList struct with the TypeMetadata initialised to the current
// version.
func NewBlockAffinityList() *BlockAffinityList {
	return &BlockAffinityList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindBlockAffinityList,
			APIVersion: GroupVersionCurrent,
		},
	}
}

func init() {
	SchemeBuilder.Register(&BlockAffinity{}, &BlockAffinityList{})
}

type IPPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              IPPoolSpec `json:"spec,omitempty"`
}

type IPPoolSpec struct {
	CIDR          string `json:"cidr" validate:"net"`
	VXLANMode     string `json:"vxlanMode,omitempty" validate:"omitempty,vxlanMode"`
	IPIPMode      string `json:"ipipMode,omitempty" validate:"omitempty,ipIpMode"`
	NATOutgoing   bool   `json:"natOutgoing,omitempty"`
	Disabled      bool   `json:"disabled,omitempty"`
	BlockSize     int    `json:"blockSize,omitempty"`
	NodeSelector  string `json:"nodeSelector,omitempty" validate:"omitempty,selector"`
	NATOutgoingV1 bool   `json:"nat-outgoing,omitempty" validate:"omitempty,mustBeFalse"`
}

type IPPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IPPool `json:"items"`
}
