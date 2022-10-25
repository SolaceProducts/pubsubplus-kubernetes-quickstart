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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventBrokerSpec defines the desired state of EventBroker
type EventBrokerSpec struct {
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Redundancy true specifies HA deployment, false specifies Non-HA
	Redundancy bool `json:"redundancy"`
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Developer true specifies minimum footprint deployment, not for production use. Overrides SystemScaling parameters.
	Developer bool `json:"developer"`
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Pod disruption budget for the broker in HA mode. For this to be `true` `solace.redundancy` has to also be `true`
	PodDisruptionBudgetForHA bool `json:"podDisruptionBudgetForHA"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// SystemScaling provides exact fine-grained specification of the event broker scaling parameters
	// and the assigned CPU / memory resources to the Pod
	SystemScaling *SystemScaling `json:"systemScaling,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// Monitoring defines Prometheus exporter for monitoring broker
	Monitoring *Monitoring `json:"monitoring,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// Service defines service details of how broker is exposed
	Service *Service `json:"service,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// Storage defines storage details of the broker
	Storage *Storage `json:"storage,omitempty"`
}

// Service defines parameters configure Service details for the Broker
type Service struct {
	//+optional
	//+kubebuilder:validation:Type:=string
	// Options to expose the broker. Options include ClusterIP, NodePort, LoadBalancer
	ServiceType v1.ServiceType `json:"type,omitempty"`
	//service.annotations	service.annotations allows to add provider-specific service annotations	Undefined
	//service.ports	Define PubSub+ service ports exposed. servicePorts are external, mapping to cluster-local pod containerPorts	initial set of frequently used ports, refer to values.yaml
}

// Storage defines parameters configure Storage details for the Broker
type Storage struct {
	//storage.persistent	false to use ephemeral storage at pod level; true to request persistent storage through a StorageClass	true, false is not recommended for production use
	//storage.slow	true to indicate slow storage used, e.g. for NFS.	false
	//storage.customVolumeMount	customVolumeMount can be used to specify a YAML fragment how the data volume should be mounted instead of using a storage class.	Undefined
	//storage.useStorageClass	Name of the StorageClass to be used to request persistent storage volumes	Undefined, meaning to use the "default" StorageClass for the Kubernetes cluster
	//storage.size	Size of the persistent storage to be used; Refer to the Solace documentation and  for storage size requirements	30Gi
	//storage.monitorStorageSize	If provided this will create and assign the minimum recommended storage to Monitor pods. For initial deployments only.	1500M
	//storage.useStorageGroup	true to use a single mount point storage-group, as recommended from PubSub+ version 9.12. Undefined or false is legacy behavior. Note: legacy mount still works for newer versions but may be deprecated in the future.	Undefined
}

type SystemScaling struct {
	// +kubebuilder:default:=100
	MaxConnections int `json:"maxConnections,omitempty"`
	// +kubebuilder:default:=100
	MaxQueueMessages int `json:"maxQueueMessages,omitempty"`
	// +kubebuilder:default:=1000
	MaxSpoolUsage       int    `json:"maxSpoolUsage,omitempty"`
	MessagingNodeCpu    string `json:"messagingNodeCpu,omitempty"`
	MessagingNodeMemory string `json:"messagingNodeMemory,omitempty"`
}

// EventBrokerStatus defines the observed state of EventBroker
type EventBrokerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// BrokerPods are the names of the eventbroker pods
	BrokerPods []string `json:"brokerpods"`
}

// Monitoring defines parameters to use Prometheus Exporter
type Monitoring struct {
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Enabled true enables the Exporter for use.
	Enabled bool `json:"enabled"`
	//+optional
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=ghcr.io/solacedev/solace_prometheus_exporter
	// Image specifies the custom image to use as prometheus exporter.
	Image string `json:"image"`
	//+optional
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=v0.0.1
	// Tag specifies the tag of the image to be used
	Tag string `json:"tag"`
	//+optional
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=Always
	// ImagePullPolicy specifies Image Pull Policy for Exporter
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// +optional
	//+kubebuilder:validation:Type:=array
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=number
	//+kubebuilder:default:=9628
	// Container Port for  Prometheus Exporter
	ContainerPort int32 `json:"port,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=number
	//+kubebuilder:default:=5
	// Timeout configuration for Prometheus Exporter scrapper
	TimeOut int32 `json:"timeOut,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Defines if Prometheus Exporter uses TLS configuration
	ListenTLS bool `json:"listenTLS,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Defines if Prometheus Exporter verifies SSL
	SSLVerify bool `json:"sslVerify,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Defines if Prometheus Exporter should include rates
	IncludeRates bool `json:"includeRates,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=ClusterIP
	// Defines if Prometheus Exporter should include rates
	ServiceType v1.ServiceType `json:"serviceType,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=eventbrokers,shortName=eb

// EventBroker is the Schema for the eventbrokers API
type EventBroker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventBrokerSpec   `json:"spec,omitempty"`
	Status EventBrokerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventBrokerList contains a list of EventBroker
type EventBrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventBroker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventBroker{}, &EventBrokerList{})
}
