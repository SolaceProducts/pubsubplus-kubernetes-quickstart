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
	"fmt"
	"strconv"
	"strings"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *PubSubPlusEventBrokerReconciler) newDeploymentForPrometheusExporter(name string, secret *corev1.Secret, m *eventbrokerv1alpha1.PubSubPlusEventBroker) *appsv1.Deployment {
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
							Name:            name,
							Image:           m.Spec.Monitoring.MonitoringImage.Repository + ":" + m.Spec.Monitoring.MonitoringImage.Tag,
							ImagePullPolicy: m.Spec.Monitoring.MonitoringImage.ImagePullPolicy,
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
									Value: fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", getPubSubPlusEventBrokerProtocol(&m.Spec), getObjectName("Service", m.Name), m.Namespace, getPubSubPlusEventBrokerPort(&m.Spec.Service, &m.Spec.BrokerTLS)),
								},
								{
									Name:  "SOLACE_LISTEN_TLS",
									Value: getExporterTLSConfiguration(&m.Spec.Monitoring),
								},
								{
									Name:  "SOLACE_USER",
									Value: "admin",
								},
								{
									Name:  "SOLACE_PASSWORD",
									Value: string(secret.Data[secretKeyName]),
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
						},
					},
					ImagePullSecrets: m.Spec.Monitoring.MonitoringImage.ImagePullSecrets,
				},
			},
		},
	}

	//Set TLS configuration
	if m.Spec.Monitoring.MonitoringMetricEndpoint != nil && len(strings.TrimSpace(m.Spec.Monitoring.MonitoringMetricEndpoint.EndpointTLSConfigSecret)) > 0 {
		allVolumes := dep.Spec.Template.Spec.Volumes
		allVolumes = append(allVolumes, corev1.Volume{
			Name: "server-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  m.Spec.Monitoring.MonitoringMetricEndpoint.EndpointTLSConfigSecret,
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
			Value: "/mnt/disks/solace/" + m.Spec.Monitoring.MonitoringMetricEndpoint.EndpointTlsConfigServerCertName,
		})
		allEnv = append(allEnv, corev1.EnvVar{
			Name:  "SOLACE_PRIVATE_KEY",
			Value: "/mnt/disks/solace/" + m.Spec.Monitoring.MonitoringMetricEndpoint.EndpointTlsConfigPrivateKeyName,
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

func (r *PubSubPlusEventBrokerReconciler) newServiceForPrometheusExporter(exporter *eventbrokerv1alpha1.Monitoring, svcName string, m *eventbrokerv1alpha1.PubSubPlusEventBroker) *corev1.Service {
	dep := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: m.Namespace,
			Labels:    getMonitoringDeploymentSelector(m.Name),
		},
		Spec: corev1.ServiceSpec{
			Type: exporter.ServiceType,
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
	// Set PubSubPlusEventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}
func getPubSubPlusEventBrokerPort(m *eventbrokerv1alpha1.Service, b *eventbrokerv1alpha1.BrokerTLS) int32 {
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

func getPubSubPlusEventBrokerProtocol(m *eventbrokerv1alpha1.EventBrokerSpec) string {
	if m.BrokerTLS.Enabled {
		return "https"
	}
	return "http"
}

func getExporterContainerPort(m *eventbrokerv1alpha1.Monitoring) int32 {
	if m.MonitoringMetricEndpoint == nil || m.MonitoringMetricEndpoint.ContainerPort == 0 {
		return 9628
	}
	return m.MonitoringMetricEndpoint.ContainerPort
}

func getExporterServicePort(m *eventbrokerv1alpha1.Monitoring) int32 {
	if m.MonitoringMetricEndpoint == nil || m.MonitoringMetricEndpoint.ServicePort == 0 {
		return 9628
	}
	return m.MonitoringMetricEndpoint.ServicePort
}

func getExporterHttpProtocolType(m *eventbrokerv1alpha1.Monitoring) string {
	if m.MonitoringMetricEndpoint != nil && len(strings.TrimSpace(m.MonitoringMetricEndpoint.Name)) > 0 {
		return m.MonitoringMetricEndpoint.Name
	} else if m.MonitoringMetricEndpoint == nil || !m.MonitoringMetricEndpoint.ListenTLS {
		return "tcp-metrics"
	} else if m.MonitoringMetricEndpoint.ListenTLS {
		return "tls-metrics"
	}
	return "metrics"
}

func getExporterServiceProtocol(m *eventbrokerv1alpha1.Monitoring) corev1.Protocol {
	if m.MonitoringMetricEndpoint == nil || m.MonitoringMetricEndpoint.Protocol == "" {
		return corev1.ProtocolTCP
	}
	return m.MonitoringMetricEndpoint.Protocol
}

func getExporterTLSConfiguration(m *eventbrokerv1alpha1.Monitoring) string {
	if m.MonitoringMetricEndpoint == nil || m.MonitoringMetricEndpoint.ListenTLS == false {
		return "false"
	}
	return strconv.FormatBool(m.MonitoringMetricEndpoint.ListenTLS)
}
