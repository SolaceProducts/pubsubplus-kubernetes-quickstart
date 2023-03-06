/*
Copyright 2023 Solace Corporation

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
	"fmt"
	"strconv"
	"strings"

	eventbrokerv1beta1 "github.com/SolaceProducts/pubsubplus-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *PubSubPlusEventBrokerReconciler) newDeploymentForPrometheusExporter(name string, monitoringSecret *corev1.Secret, m *eventbrokerv1beta1.PubSubPlusEventBroker) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: getMonitoringDeploymentSelector(m.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getMonitoringDeploymentSelector(m.Name),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "exporter",
							Image:           getExporterImageDetails(m.Spec.Monitoring.MonitoringImage),
							ImagePullPolicy: getExporterImagePullPolicy(m.Spec.Monitoring.MonitoringImage),
							Ports: []corev1.ContainerPort{{
								Name:          getExporterHttpProtocolType(&m.Spec.Monitoring),
								ContainerPort: getExporterContainerPort(&m.Spec.Monitoring),
							}},

							Env: []corev1.EnvVar{
								{
									Name:  "SOLACE_WEB_LISTEN_ADDRESS",
									Value: fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", getExporterHttpProtocolType(&m.Spec.Monitoring), getObjectName("PrometheusExporterService", m.Name), m.Namespace, getExporterContainerPort(&m.Spec.Monitoring)),
								},
								{
									Name:  "SOLACE_SCRAPE_URI",
									Value: fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", getPubSubPlusEventBrokerProtocol(&m.Spec), getObjectName("BrokerService", m.Name), m.Namespace, getPubSubPlusEventBrokerPort(&m.Spec.Service, &m.Spec.BrokerTLS)),
								},
								{
									Name:  "SOLACE_LISTEN_TLS",
									Value: getExporterTLSConfiguration(&m.Spec.Monitoring),
								},
								{
									Name:  "SOLACE_USERNAME",
									Value: "monitor",
								},
								{
									Name: "SOLACE_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: monitoringSecret.Name,
											},
											Key: monitorSecretKeyName,
										},
									},
								},
								{
									Name:  "SOLACE_SCRAPE_TIMEOUT",
									Value: fmt.Sprint(m.Spec.Monitoring.TimeOut) + "s",
								},
								{
									Name:  "SOLACE_SSL_VERIFY",
									Value: strconv.FormatBool(m.Spec.Monitoring.SSLVerify),
								},
								{
									Name:  "SOLACE_INCLUDE_RATES",
									Value: strconv.FormatBool(m.Spec.Monitoring.IncludeRates),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &[]bool{false}[0], // Set to false
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										corev1.Capability("ALL"),
									},
								},
								RunAsNonRoot:             &[]bool{true}[0],  // Set to true
								AllowPrivilegeEscalation: &[]bool{false}[0], // Set to false
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
					},
					ImagePullSecrets: getExporterImagePullSecrets(m.Spec.Monitoring.MonitoringImage),
				},
			},
		},
	}

	// OpenShift and namespace must be considered for RunAsUser
	// Only set it if not on OpenShift or using the "default" namespace
	// otherwise leave it undefined
	if !r.IsOpenShift || m.Namespace == corev1.NamespaceDefault {
		dep.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser = &[]int64{10001}[0]
	}

	//Set TLS configuration
	if m.Spec.Monitoring.MonitoringMetricsEndpoint != nil && len(strings.TrimSpace(m.Spec.Monitoring.MonitoringMetricsEndpoint.EndpointTLSConfigSecret)) > 0 {
		allVolumes := dep.Spec.Template.Spec.Volumes
		allVolumes = append(allVolumes, corev1.Volume{
			Name: "server-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  m.Spec.Monitoring.MonitoringMetricsEndpoint.EndpointTLSConfigSecret,
					DefaultMode: &[]int32{0400}[0],
				},
			},
		})
		allContainerVolumeMounts := dep.Spec.Template.Spec.Containers[0].VolumeMounts
		allContainerVolumeMounts = append(allContainerVolumeMounts, corev1.VolumeMount{
			Name:      "server-certs",
			MountPath: "/mnt/disks/solace",
			ReadOnly:  true,
		})
		allEnv := dep.Spec.Template.Spec.Containers[0].Env

		allEnv = append(allEnv, corev1.EnvVar{
			Name:  "SOLACE_SERVER_CERT",
			Value: "/mnt/disks/solace/" + m.Spec.Monitoring.MonitoringMetricsEndpoint.EndpointTlsConfigServerCertName,
		})
		allEnv = append(allEnv, corev1.EnvVar{
			Name:  "SOLACE_PRIVATE_KEY",
			Value: "/mnt/disks/solace/" + m.Spec.Monitoring.MonitoringMetricsEndpoint.EndpointTlsConfigPrivateKeyName,
		})

		dep.Spec.Template.Spec.Containers[0].Env = allEnv
		dep.Spec.Template.Spec.Volumes = allVolumes
		dep.Spec.Template.Spec.Containers[0].VolumeMounts = allContainerVolumeMounts
	} else {
		allEnv := dep.Spec.Template.Spec.Containers[0].Env
		allEnv = append(allEnv, corev1.EnvVar{
			Name:  "SOLACE_SERVER_CERT",
			Value: ".", //This is a mandatory parameter.
		})
		allEnv = append(allEnv, corev1.EnvVar{
			Name:  "SOLACE_PRIVATE_KEY",
			Value: ".", //This is a mandatory parameter.
		})
		dep.Spec.Template.Spec.Containers[0].Env = allEnv
	}

	// Set PubSubPlusEventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

func (r *PubSubPlusEventBrokerReconciler) newServiceForPrometheusExporter(exporter *eventbrokerv1beta1.Monitoring, svcName string, m *eventbrokerv1beta1.PubSubPlusEventBroker) *corev1.Service {
	dep := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: m.Namespace,
			Labels:    getMonitoringDeploymentSelector(m.Name),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       getExporterHttpProtocolType(&m.Spec.Monitoring),
					Protocol:   getExporterServiceProtocol(&m.Spec.Monitoring),
					Port:       getExporterServicePort(exporter),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: getExporterContainerPort(exporter)},
				},
			},
			Selector: getMonitoringDeploymentSelector(m.Name),
		},
	}
	if exporter.MonitoringMetricsEndpoint != nil && exporter.MonitoringMetricsEndpoint.ServiceType != "" {
		dep.Spec.Type = exporter.MonitoringMetricsEndpoint.ServiceType
	} else {
		dep.Spec.Type = corev1.ServiceTypeClusterIP
	}
	// Set PubSubPlusEventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}
func getPubSubPlusEventBrokerPort(m *eventbrokerv1beta1.Service, b *eventbrokerv1beta1.BrokerTLS) int32 {
	if len(m.Ports) == 0 {
		return 8080
	}
	for i := range m.Ports {
		if b.Enabled {
			if m.Ports[i].Name == tlsSempPortName {
				return m.Ports[i].ContainerPort
			}
		} else {
			if m.Ports[i].Name == tcpSempPortName {
				return m.Ports[i].ContainerPort
			}
		}
	}
	return 0
}

func getPubSubPlusEventBrokerProtocol(m *eventbrokerv1beta1.EventBrokerSpec) string {
	if m.BrokerTLS.Enabled {
		return "https"
	}
	return "http"
}

func getExporterContainerPort(m *eventbrokerv1beta1.Monitoring) int32 {
	if m.MonitoringMetricsEndpoint == nil || m.MonitoringMetricsEndpoint.ContainerPort == 0 {
		return 9628
	}
	return m.MonitoringMetricsEndpoint.ContainerPort
}

func getExporterServicePort(m *eventbrokerv1beta1.Monitoring) int32 {
	if m.MonitoringMetricsEndpoint == nil || m.MonitoringMetricsEndpoint.ServicePort == 0 {
		return 9628
	}
	return m.MonitoringMetricsEndpoint.ServicePort
}

func getExporterHttpProtocolType(m *eventbrokerv1beta1.Monitoring) string {
	if m.MonitoringMetricsEndpoint != nil && len(strings.TrimSpace(m.MonitoringMetricsEndpoint.Name)) > 0 {
		return m.MonitoringMetricsEndpoint.Name
	} else if m.MonitoringMetricsEndpoint != nil && m.MonitoringMetricsEndpoint.ListenTLS {
		return "tls-metrics"
	}
	return "tcp-metrics"
}

func getExporterServiceProtocol(m *eventbrokerv1beta1.Monitoring) corev1.Protocol {
	if m.MonitoringMetricsEndpoint == nil || m.MonitoringMetricsEndpoint.Protocol == "" {
		return corev1.ProtocolTCP
	}
	return m.MonitoringMetricsEndpoint.Protocol
}

func getExporterTLSConfiguration(m *eventbrokerv1beta1.Monitoring) string {
	if m.MonitoringMetricsEndpoint == nil || !m.MonitoringMetricsEndpoint.ListenTLS {
		return "false"
	}
	return strconv.FormatBool(m.MonitoringMetricsEndpoint.ListenTLS)
}

func getExporterImageDetails(bm *eventbrokerv1beta1.MonitoringImage) string {
	imageRepo := "ghcr.io/solacedev/solace_prometheus_exporter"
	imageTag := "latest"

	if bm != nil && len(strings.TrimSpace(bm.Repository)) > 0 {
		imageRepo = bm.Repository
	}
	if bm != nil && len(strings.TrimSpace(bm.Tag)) > 0 {
		imageTag = bm.Tag
	}
	return imageRepo + ":" + imageTag
}

func getExporterImagePullPolicy(bm *eventbrokerv1beta1.MonitoringImage) corev1.PullPolicy {
	imagePullPolicy := corev1.PullIfNotPresent
	if bm != nil && len(bm.ImagePullPolicy) > 0 {
		imagePullPolicy = bm.ImagePullPolicy
	}
	return imagePullPolicy
}

func getExporterImagePullSecrets(bm *eventbrokerv1beta1.MonitoringImage) []corev1.LocalObjectReference {
	var imagePullSecret []corev1.LocalObjectReference
	if bm != nil && len(bm.ImagePullSecrets) > 0 {
		imagePullSecret = bm.ImagePullSecrets
	}
	return imagePullSecret
}
