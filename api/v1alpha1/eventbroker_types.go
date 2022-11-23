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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventBrokerSpec defines the desired state of PubSubPlusEventBroker
type EventBrokerSpec struct {
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Redundancy true specifies HA deployment, false specifies Non-HA.
	Redundancy bool `json:"redundancy"`
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Developer true specifies a minimum footprint scaled-down deployment, not for production use.
	// If set to true it overrides SystemScaling parameters.
	Developer bool `json:"developer"`
	//+optional
	//+kubebuilder:validation:Type:=array
	// List of extra environment variables to be added to the PubSubPlusEventBroker container.
	// A primary use case is to specify configuration keys, although the variables defined here will not override the ones defined in ConfigMap
	ExtraEnvVars []*ExtraEnvVar `json:"extraEnvVars"`
	//+optional
	//+kubebuilder:validation:Type:=string
	// List of extra environment variables to be added to the PubSubPlusEventBroker container from an existing ConfigMap
	ExtraEnvVarsCM string `json:"extraEnvVarsCM,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=string
	// List of extra environment variables to be added to the PubSubPlusEventBroker container from an existing Secret
	ExtraEnvVarsSecret string `json:"extraEnvVarsSecret,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// PodDisruptionBudgetForHA enables setting up PodDisruptionBudget for the broker pods in HA deployment.
	// This parameter is ignored for non-HA deployments (if redundancy is false).
	PodDisruptionBudgetForHA bool `json:"podDisruptionBudgetForHA"`
	//+optional
	//+kubebuilder:validation:Enum=automatedRolling;manualPodRestart
	//+kubebuilder:default:=automatedRolling
	// UpdateStrategy specifies how to update an existing deployment. manualPodRestart waits for user intervention.
	UpdateStrategy PubSubPlusEventBrokerUpdateStrategy `json:"updateStrategy"`
	//+optional
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:="UTC"
	// Defines the timezone for the event broker container, if undefined default is UTC. Valid values are tz database time zone names.
	Timezone string `json:"timezone"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// SystemScaling provides exact fine-grained specification of the event broker scaling parameters
	// and the assigned CPU / memory resources to the Pod.
	SystemScaling *SystemScaling `json:"systemScaling,omitempty"`
	//+kubebuilder:validation:Type:=object
	// Image defines container image parameters for the event broker.
	BrokerImage *BrokerImage `json:"image,omitempty"`
	//+kubebuilder:validation:Type:=object
	// PodSecurityContext defines the pod security context for the event broker.
	PodSecurityContext *PodSecurityContext `json:"securityContext,omitempty"`
	//+kubebuilder:validation:Type:=object
	// ServiceAccount defines a ServiceAccount dedicated to the PubSubPlusEventBroker
	ServiceAccount BrokerServiceAccount `json:"serviceAccount,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// TLS provides TLS configuration for the event broker.
	BrokerTLS *BrokerTLS `json:"tls,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// Service defines broker service details.
	Service *Service `json:"service,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// Storage defines storage details for the broker.
	Storage *Storage `json:"storage,omitempty"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// Monitoring specifies a Prometheus monitoring endpoint for the event broker
	Monitoring *Monitoring `json:"monitoring,omitempty"`
}

type PubSubPlusEventBrokerUpdateStrategy string

const (
	AutomatedRollingUpdateStrategy PubSubPlusEventBrokerUpdateStrategy = "automatedRolling"
	manualPodRestartUpdateStrategy PubSubPlusEventBrokerUpdateStrategy = "manualPodRestart"
)

// Port defines parameters configure Service details for the Broker
type BrokerPort struct {
	//+kubebuilder:validation:Type:=string
	// Unique name for the port that can be referred to by services.
	Name string `json:"name"`
	//+kubebuilder:validation:Type:=string
	// Protocol for port. Must be UDP, TCP, or SCTP.
	Protocol corev1.Protocol `json:"protocol"`
	//+kubebuilder:validation:Type:=number
	// Number of port to expose on the pod.
	ContainerPort int32 `json:"containerPort"`
	//+kubebuilder:validation:Type:=number
	// Number of port to expose on the service
	ServicePort int32 `json:"servicePort"`
}

// Service defines parameters configure Service details for the Broker
type Service struct {
	//+optional
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=LoadBalancer
	// ServiceType specifies how to expose the broker services. Options include ClusterIP, NodePort, LoadBalancer (default).
	ServiceType corev1.ServiceType `json:"type"`
	//+optional
	//+kubebuilder:validation:Type:=object
	//+kubebuilder:default:={}
	// Annotations allows adding provider-specific service annotations
	Annotations map[string]string `json:"annotations"`
	//+optional
	//+kubebuilder:validation:Type:=array
	//+kubebuilder:default:={{name:"tcp-ssh",protocol:"TCP",containerPort:2222,servicePort:2222},{name:"tcp-semp",protocol:"TCP",containerPort:8080,servicePort:8080},{name:"tls-semp",protocol:"TCP",containerPort:1943,servicePort:1943},{name:"tcp-smf",protocol:"TCP",containerPort:55555,servicePort:55555},{name:"tcp-smfcomp",protocol:"TCP",containerPort:55003,servicePort:55003},{name:"tls-smf",protocol:"TCP",containerPort:55443,servicePort:55443},{name:"tcp-smfroute",protocol:"TCP",containerPort:55556,servicePort:55556},{name:"tcp-web",protocol:"TCP",containerPort:8008,servicePort:8008},{name:"tls-web",protocol:"TCP",containerPort:1443,servicePort:1443},{name:"tcp-rest",protocol:"TCP",containerPort:9000,servicePort:9000},{name:"tls-rest",protocol:"TCP",containerPort:9443,servicePort:9443},{name:"tcp-amqp",protocol:"TCP",containerPort:5672,servicePort:5672},{name:"tls-amqp",protocol:"TCP",containerPort:5671,servicePort:5671},{name:"tcp-mqtt",protocol:"TCP",containerPort:1883,servicePort:1883},{name:"tls-mqtt",protocol:"TCP",containerPort:8883,servicePort:8883},{name:"tcp-mqttweb",protocol:"TCP",containerPort:8000,servicePort:8000},{name:"tls-mqttweb",protocol:"TCP",containerPort:8443,servicePort:8443}}
	// Ports specifies the ports to expose PubSubPlusEventBroker services.
	Ports []*BrokerPort `json:"ports"`
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
	MaxSpoolUsage int `json:"maxSpoolUsage,omitempty"`
	// +kubebuilder:default:="2"
	MessagingNodeCpu string `json:"messagingNodeCpu,omitempty"`
	// +kubebuilder:default:="4025Mi"
	MessagingNodeMemory string `json:"messagingNodeMemory,omitempty"`
}

// EventBrokerStatus defines the observed state of the event PubSubPlusEventBroker
type EventBrokerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// BrokerPods are the names of the eventbroker pods
	BrokerPods []string `json:"brokerpods"`
}

// BrokerTLS defines TLS configuration for the PubSubPlusEventBroker
type BrokerTLS struct {
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Enabled true enables TLS for the broker.
	Enabled bool `json:"enabled"`
	//+optional
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=example-tls-secret
	// Specifies the tls configuration secret to be used for the broker
	ServerTLsConfigSecret string `json:"serverTlsConfigSecret"`
	//+optional
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=tls.key
	// Name of the Certificate file in the `serverCertificatesSecret`
	TLSCertName string `json:"certFilename"`
	//+optional
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=tls.crt
	// Name of the Key file in the `serverCertificatesSecret`
	TLSCertKeyName string `json:"certKeyFilename"`
}

// ServiceAccount defines a ServiceAccount dedicated to the PubSubPlusEventBroker
type BrokerServiceAccount struct {
	//+kubebuilder:validation:Type:=string
	// Name specifies the name of an existing ServiceAccount dedicated to the PubSubPlusEventBroker.
	// If this value is missing a new ServiceAccount will be created.
	Name string `json:"name"`
}

// ExtraEnvVar defines environment variables to be added to the PubSubPlusEventBroker container
type ExtraEnvVar struct {
	//+kubebuilder:validation:Type:=string
	// Specifies the Name of an environment variable to be added to the PubSubPlusEventBroker container
	Name string `json:"name"`
	//+kubebuilder:validation:Type:=string
	// Specifies the Value of an environment variable to be added to the PubSubPlusEventBroker container
	Value string `json:"value"`
}

// BrokerImage defines Image details and pulling configurations
type BrokerImage struct {
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:="solace/solace-pubsub-standard"
	// Defines the container image repo where the event broker image is pulled from
	Repository string `json:"repository,omitempty"`
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:="latest"
	// Specifies the tag of the container image to be used for the event broker.
	Tag string `json:"tag,omitempty"`
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:="IfNotPresent"
	// Specifies ImagePullPolicy of the container image for the event broker.
	ImagePullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
	//+kubebuilder:validation:Type:=array
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	ImagePullSecrets []corev1.LocalObjectReference `json:"pullSecretName,omitempty"`
}

// PodSecurityContext defines the pod security context for the PubSubPlusEventBroker
type PodSecurityContext struct {
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=true
	// Enabled true will enable the Pod Security Context.
	Enabled bool `json:"enabled"`
	//+optional
	//+kubebuilder:validation:Type:=number
	//+kubebuilder:default:=1000002
	// Specifies fsGroup in pod security context.
	FSGroup int64 `json:"fsGroup"`
	//+optional
	//+kubebuilder:validation:Type:=number
	//+kubebuilder:default:=1000001
	// Specifies runAsUser in pod security context.
	RunAsUser int64 `json:"runAsUser"`
}

// MonitoringImage defines Image details and pulling configurations for the Prometheus Exporter for Monitoring
type MonitoringImage struct {
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=ghcr.io/solacedev/solace_prometheus_exporter
	// Defines the container image repo where the Prometheus Exporter image is pulled from
	Repository string `json:"repository,omitempty"`
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=latest
	// Specifies the tag of the container image to be used for the Prometheus Exporter.
	Tag string `json:"tag,omitempty"`
	//+kubebuilder:validation:Type:=string
	//+kubebuilder:default:=IfNotPresent
	// Specifies ImagePullPolicy of the container image for the Prometheus Exporter.
	ImagePullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// +optional
	//+kubebuilder:validation:Type:=array
	ImagePullSecrets []corev1.LocalObjectReference `json:"pullSecretName,omitempty"`
}

// Monitoring defines parameters to use Prometheus Exporter
type Monitoring struct {
	//+optional
	//+kubebuilder:validation:Type:=boolean
	//+kubebuilder:default:=false
	// Enabled true enables the setup of the Prometheus Exporter.
	Enabled bool `json:"enabled"`
	//+optional
	//+kubebuilder:validation:Type:=object
	// Image defines container image parameters for the Prometheus Exporter.
	MonitoringImage *MonitoringImage `json:"image,omitempty"`
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
	//+kubebuilder:validation:Type:=string
	// Defines TLS secret name to set up TLS configuration
	TLSSecret string `json:"tlsSecretName,omitempty"`
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
	// Defines the service type for Prometheus Exporter
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=pubsubpluseventbrokers,shortName=eb;eventbroker

// PubSubPlusEventBroker is the Schema for the pubsubpluseventbrokers API
type PubSubPlusEventBroker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventBrokerSpec   `json:"spec,omitempty"`
	Status EventBrokerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PubSubPlusEventBrokerList contains a list of PubSubPlusEventBroker
type PubSubPlusEventBrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PubSubPlusEventBroker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PubSubPlusEventBroker{}, &PubSubPlusEventBrokerList{})
}
