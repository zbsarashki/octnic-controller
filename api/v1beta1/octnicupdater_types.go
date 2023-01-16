/*
Copyright 2023 tbc project.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OctNicUpdaterSpec defines the desired state of OctNicUpdater
type OctNicUpdaterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Acclr     string `json:"acclr,omitempty"`
	OSImage string `json:"osimage,omitempty"`
	FWImage string `json:"fwimage,omitempty"`
	NodeName  string `json:"nodename,omitempty"`
	// Device configuration
	PciAddr   string `json:"pciAddr,omitempty"`
	NumVfs string `json:"numvfs,omitempty"`
	/* 
	Pass resource names and their mappings through CRD
	Syntax:
	resourcename:
	  - "marvell_sriov_net_vamp#0"
	  - "marvell_sriov_net_rmp#8-15"
	  - "marvell_sriov_net_dip#20-21"
	  - "marvell_sriov_net_dpp#32,36-37,40-47"
	*/
	ResourceName []string `json:"resourceName,omitempty"`
	ResourcePrefix  string `json:"resourcePrefix,omitempty"`
	// To be removed once support for checking OS,and FW versions is added
	// to the tools image. In the absence of support from the tools image
	// for checking the required runtime versions on the device, the
	// Operation field in the CRD can be used to request device Update.
	// In this case, a URL passed to the operator at helm install time, will 
	// be passed to the tools image. The tools image will download and apply 
	// any and all update images ($URL/$OSImage, $URL/$FWImage) that it
	// finds at URL. Upon completion of the update (The update pod's state
	// is completed), the CRD Field Operation will be modified by the 
	// Operator, and changed to Run.
	// The values the Operation field takes are: RUN, MAINTENANCE.
	Operation string `json:"operation,omitempty"`
}

type OctNicOperationState string

const (
	// Unknown state
	OctS0 OctNicOperationState = "Unknown"
	// Driver Loaded
	OctS1 OctNicOperationState = "DriverLoaded"
	// Driver Validate
	OctS2 OctNicOperationState = "DriverValidate"
	// DP Loaded
	OctS3 OctNicOperationState = "DpLoaded"
	// DP Validate
	OctS4 OctNicOperationState = "DpValidate"
	// Run
	OctS5 OctNicOperationState = "Run"
	// Maintenance
	OctM0 OctNicOperationState = "UpdateRequest"
	// Maintenance
	OctM1 OctNicOperationState = "UpdateInProgress"
)

type OctNicDevice struct {
	NodeName string               `json:"osversion,omitempty"`
	PciAddr  string               `json:"pciAddr,omitempty"`
	OpState  OctNicOperationState `json:"opstate,omitempty"`
}

// OctNicUpdaterStatus defines the observed state of OctNicUpdater
type OctNicUpdaterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	OctNics []OctNicDevice `json:"octnic,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OctNicUpdater is the Schema for the octnicupdaters API
type OctNicUpdater struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OctNicUpdaterSpec   `json:"spec,omitempty"`
	Status OctNicUpdaterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OctNicUpdaterList contains a list of OctNicUpdater
type OctNicUpdaterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OctNicUpdater `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OctNicUpdater{}, &OctNicUpdaterList{})
}
