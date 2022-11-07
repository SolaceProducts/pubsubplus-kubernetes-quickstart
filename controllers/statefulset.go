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

package controllers

import (
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
)

// statefulsetForEventBroker returns a new pubsubpluseventbroker StatefulSet object
func (r *PubSubPlusEventBrokerReconciler) createStatefulsetForEventBroker(stsName string, m *eventbrokerv1alpha1.PubSubPlusEventBroker, sa *corev1.ServiceAccount) *appsv1.StatefulSet {
	nodeType := getBrokerNodeType(stsName)

	// Determine broker sizing
	var storageSize string
	if nodeType == "monitor" {
		storageSize = "3Gi"
	} else {
		storageSize = (map[bool]string{true: "7Gi", false: "17Gi"})[m.Spec.Developer]
	}
	dep := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		// Followings are immutable fields of the StatefulSet - cannot be part of the update
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{ // Refers to the broker Pod labels - see template below
				MatchLabels: getPodLabels(m.Name, nodeType),
			},
			Replicas:                             &[]int32{1}[0], // Set to 1
			ServiceName:                          getObjectName("DiscoveryService", m.Name),
			PodManagementPolicy:                  "",
			UpdateStrategy:                       appsv1.StatefulSetUpdateStrategy{},
			RevisionHistoryLimit:                 new(int32),
			MinReadySeconds:                      0,
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resource.MustParse(storageSize),
							},
						},
					},
				},
			},
		},
	}

	r.updateStatefulsetForEventBroker(stsName, m, dep, sa)
	// Set PubSubPlusEventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// statefulsetForEventBroker returns an updated pubsubpluseventbroker StatefulSet object
func (r *PubSubPlusEventBrokerReconciler) updateStatefulsetForEventBroker(stsName string, m *eventbrokerv1alpha1.PubSubPlusEventBroker, dep *appsv1.StatefulSet, sa *corev1.ServiceAccount) {
	brokerServicesName := getObjectName("Service", m.Name)
	secretName := getObjectName("Secret", m.Name)
	configmapName := getObjectName("ConfigMap", m.Name)
	haDeployment := m.Spec.Redundancy
	nodeType := getBrokerNodeType(stsName)

	// Determine broker sizing
	var cpuRequests, cpuLimits string
	var memRequests, memLimits string
	var maxConnections, maxQueueMessages, maxSpoolUsage int
	// TODO: _types.go has already defaults. Review if those indeed need to be duplicated here.
	if nodeType == "monitor" {
		cpuRequests = "1"
		cpuLimits = "1"
		memRequests = "2Gi"
		memLimits = "2Gi"
		maxConnections = 100
		maxQueueMessages = 100
		maxSpoolUsage = 1000
	} else {
		// First determine default settings for the message routing broker nodes, depending on developer mode set
		// refer to https://docs.solace.com/Admin-Ref/Resource-Calculator/pubsubplus-resource-calculator.html
		cpuRequests = (map[bool]string{true: "1", false: "2"})[m.Spec.Developer]
		cpuLimits = (map[bool]string{true: "2", false: "2"})[m.Spec.Developer]
		memRequests = (map[bool]string{true: "3410Mi", false: "4025Mi"})[m.Spec.Developer]
		memLimits = (map[bool]string{true: "3410Mi", false: "4025Mi"})[m.Spec.Developer]
		maxConnections = (map[bool]int{true: 100, false: 100})[m.Spec.Developer]
		maxQueueMessages = (map[bool]int{true: 100, false: 100})[m.Spec.Developer]
		maxSpoolUsage = (map[bool]int{true: 1000, false: 10000})[m.Spec.Developer]
		// Overwrite for any values defined in spec.systemScaling
		if m.Spec.SystemScaling != nil && !m.Spec.Developer {
			if m.Spec.SystemScaling.MessagingNodeCpu != "" {
				cpuRequests = m.Spec.SystemScaling.MessagingNodeCpu
				cpuLimits = cpuRequests
			}
			if m.Spec.SystemScaling.MessagingNodeMemory != "" {
				memRequests = m.Spec.SystemScaling.MessagingNodeMemory
				memLimits = memRequests
			}
			if m.Spec.SystemScaling.MaxConnections > 0 {
				maxConnections = m.Spec.SystemScaling.MaxConnections
			}
			if m.Spec.SystemScaling.MaxQueueMessages > 0 {
				maxQueueMessages = m.Spec.SystemScaling.MaxQueueMessages
			}
			if m.Spec.SystemScaling.MaxSpoolUsage > 0 {
				maxSpoolUsage = m.Spec.SystemScaling.MaxSpoolUsage
			}
		}
	}

	// Update fields
	dep.Spec.UpdateStrategy = appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.OnDeleteStatefulSetStrategyType,
	}
	dep.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: getPodLabels(m.Name, nodeType),
			// Note the resource version of upstream objects
			// TODO: Consider https://github.com/banzaicloud/k8s-objectmatcher
			Annotations: map[string]string{
				dependenciesSignatureAnnotationName: hash(m.Spec),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "pubsubplus",
					Image:           m.Spec.BrokerImage.Repository + ":" + m.Spec.BrokerImage.Tag,
					ImagePullPolicy: m.Spec.BrokerImage.ImagePullPolicy,
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse(cpuLimits),
							corev1.ResourceMemory: resource.MustParse(memLimits),
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse(cpuRequests),
							corev1.ResourceMemory: resource.MustParse(memRequests),
						},
					},
					Command: []string{
						"bash",
						"-ec",
						"source /mnt/disks/solace/init.sh\nnohup /mnt/disks/solace/startup-broker.sh &\n/usr/sbin/boot.sh",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "STATEFULSET_NAME",
							Value: stsName,
						},
						{
							Name: "STATEFULSET_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
						{
							Name:  "BROKERSERVICES_NAME",
							Value: brokerServicesName,
						},
						{
							Name:  "BROKER_MAXCONNECTIONCOUNT",
							Value: strconv.Itoa(maxConnections),
						},
						{
							Name:  "BROKER_MAXQUEUEMESSAGECOUNT",
							Value: strconv.Itoa(maxQueueMessages),
						},
						{
							Name:  "BROKER_MAXSPOOLUSAGE",
							Value: strconv.Itoa(maxSpoolUsage),
						},
						{
							Name:  "BROKER_TLS_ENEBLED",
							Value: strconv.FormatBool(m.Spec.BrokerTLS.Enabled),
						},
						{
							Name:  "BROKER_CERT_FILENAME",
							Value: m.Spec.BrokerTLS.TLSCertName,
						},
						{
							Name:  "BROKER_CERTKEY_FILENAME",
							Value: m.Spec.BrokerTLS.TLSCertKeyName,
						},
						{
							Name:  "BROKER_REDUNDANCY",
							Value: strconv.FormatBool(haDeployment),
						},
						{
							Name:  "TZ",
							Value: ":/usr/share/zoneinfo/" + m.Spec.Timezone,
						},
						{
							Name:  "UMASK",
							Value: "0022",
						},
					},
					// EnvFrom:                  []corev1.EnvFromSource{},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.IntOrString{Type: intstr.Int, IntVal: int32(8080)},
							},
						},
						InitialDelaySeconds: 300,
						TimeoutSeconds:      5,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"/mnt/disks/solace/readiness_check.sh",
								},
							},
						},
						InitialDelaySeconds: 30,
						TimeoutSeconds:      1,
						PeriodSeconds:       5,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
					Lifecycle: &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"bash",
									"-ec",
									"while ! pgrep solacedaemon ; do sleep 1; done\nkillall solacedaemon;\nwhile [ ! -d /usr/sw/var/db.upgrade ]; do sleep 1; done;",
								},
							},
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &[]bool{false}[0], // Set to false
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "podinfo",
							MountPath: "/etc/podinfo",
						},
						{
							Name:      "config-map",
							MountPath: "/mnt/disks/solace",
						},
						{
							Name:      "secrets",
							ReadOnly:  true,
							MountPath: "/mnt/disks/secrets",
						},
						{
							Name:      "dshm",
							MountPath: "/dev/shm",
						},
						{
							Name:      "data",
							MountPath: "/var/lib/solace",
						},
					},
				},
			},
			RestartPolicy:                 corev1.RestartPolicyAlways,
			TerminationGracePeriodSeconds: &[]int64{1200}[0], // 1200
			ServiceAccountName:            sa.Name,
			// NodeName:                      "",
			Volumes: []corev1.Volume{
				{
					Name: "podinfo",
					VolumeSource: corev1.VolumeSource{
						DownwardAPI: &corev1.DownwardAPIVolumeSource{
							Items: []corev1.DownwardAPIVolumeFile{
								{
									Path: "labels",
									FieldRef: &corev1.ObjectFieldSelector{
										APIVersion: "v1",
										FieldPath:  "metadata.labels",
									},
								},
							},
							DefaultMode: &[]int32{420}[0], // 420
						},
					},
				},
				{
					Name: "config-map",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configmapName,
							},
							DefaultMode: &[]int32{493}[0], // 493
						},
					},
				},
				{
					Name: "secrets",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  secretName,
							DefaultMode: &[]int32{256}[0], // 256
						},
					},
				},
				{
					Name: "dshm",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							Medium: corev1.StorageMediumMemory,
						},
					},
				},
			},
			ImagePullSecrets: m.Spec.BrokerImage.ImagePullSecrets,
			// NodeSelector:                  map[string]string{},
			// Affinity:                      &corev1.Affinity{},
			// SchedulerName:                 "",
			// Tolerations:                   []corev1.Toleration{},
			// TopologySpreadConstraints:     []corev1.TopologySpreadConstraint{},
		},
	}

	//Set Pod Security Context if Enabled
	if m.Spec.PodSecurityContext.Enabled {
		dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser: &m.Spec.PodSecurityContext.RunAsUser,
			FSGroup:   &m.Spec.PodSecurityContext.FSGroup,
		}
	}

	//Set TLS configuration
	if m.Spec.BrokerTLS.Enabled {
		allVolumes := dep.Spec.Template.Spec.Volumes
		allVolumes = append(allVolumes, corev1.Volume{
			Name: "server-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  m.Spec.BrokerTLS.ServerTLsConfigSecret,
					DefaultMode: &[]int32{0400}[0],
				},
			},
		})
		allContainerVolumeMounts := dep.Spec.Template.Spec.Containers[0].VolumeMounts
		allContainerVolumeMounts = append(allContainerVolumeMounts, corev1.VolumeMount{
			Name:      "server-certs",
			MountPath: "/mnt/disks/certs/server",
			ReadOnly:  true,
		})
		dep.Spec.Template.Spec.Volumes = allVolumes
		dep.Spec.Template.Spec.Containers[0].VolumeMounts = allContainerVolumeMounts
	}

	//Set Service Port configuration
	if len(m.Spec.Service.Ports) > 0 {
		ports := make([]corev1.ContainerPort, len(m.Spec.Service.Ports))
		for idx, pbPort := range m.Spec.Service.Ports {
			ports[idx] = corev1.ContainerPort{
				Name:          pbPort.Name,
				Protocol:      pbPort.Protocol,
				ContainerPort: pbPort.ContainerPort,
			}
		}

		dep.Spec.Template.Spec.Containers[0].Ports = ports
	}

	//Set Extra environment variables
	if len(m.Spec.ExtraEnvVars) > 0 {
		allEnv := dep.Spec.Template.Spec.Containers[0].Env
		for _, envV := range m.Spec.ExtraEnvVars {
			allEnv = append(allEnv, corev1.EnvVar{
				Name:  envV.Name,
				Value: envV.Value,
			})
		}
		dep.Spec.Template.Spec.Containers[0].Env = allEnv
	}

	allEnvFrom := []corev1.EnvFromSource{}
	//Set Extra secret environment variables
	if len(strings.TrimSpace(m.Spec.ExtraEnvVarsSecret)) > 0 {
		allEnvFrom = append(allEnvFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: m.Spec.ExtraEnvVarsSecret,
				},
			},
		})
	}

	//Set Extra configmap environment variables
	if len(strings.TrimSpace(m.Spec.ExtraEnvVarsCM)) > 0 {
		allEnvFrom = append(allEnvFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: m.Spec.ExtraEnvVarsCM,
				},
			},
		})
	}

	dep.Spec.Template.Spec.Containers[0].EnvFrom = allEnvFrom
}
