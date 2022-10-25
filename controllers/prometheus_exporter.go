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
	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
)

func (r *EventBrokerReconciler) newDeploymentForPrometheusExporter(name string, secret *corev1.Secret, broker *eventbrokerv1alpha1.EventBroker) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: broker.Namespace,
			Labels:    getBrokerPodSelector(broker.Name, PrometheusExporter),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: getBrokerPodSelector(broker.Name, PrometheusExporter),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getBrokerPodSelector(broker.Name, PrometheusExporter),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           broker.Spec.Monitoring.Image + ":" + broker.Spec.Monitoring.Tag,
							ImagePullPolicy: broker.Spec.Monitoring.ImagePullPolicy,
							Ports: []corev1.ContainerPort{{
								Name:          "http",
								ContainerPort: getPrometheusExporterPort(broker.Spec.Monitoring),
							}},

							Env: []corev1.EnvVar{
								{
									Name:  "SOLACE_WEB_LISTEN_ADDRESS",
									Value: fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", isSSLVerify(broker.Spec.Monitoring), getObjectName("PrometheusExporterService", broker.Name), broker.Namespace, getPrometheusExporterPort(broker.Spec.Monitoring)),
								},
								{
									Name:  "SOLACE_SCRAPE_URI", //hard code broker port for now.
									Value: fmt.Sprintf("%s://%s.%s.svc.cluster.local:8080", isSSLVerify(broker.Spec.Monitoring), getObjectName("DiscoveryService", broker.Name), broker.Namespace),
								},
								{
									Name:  "SOLACE_LISTEN_TLS",
									Value: strconv.FormatBool(broker.Spec.Monitoring.ListenTLS),
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
									Value: fmt.Sprint(broker.Spec.Monitoring.TimeOut, 10) + "s",
								},
								{
									Name:  "SOLACE_SSL_VERIFY",
									Value: strconv.FormatBool(broker.Spec.Monitoring.SSLVerify),
								},
								{
									Name:  "SOLACE_INCLUDE_RATES",
									Value: strconv.FormatBool(broker.Spec.Monitoring.IncludeRates),
								},
								{
									Name:  "SOLACE_SERVER_CERT", //hard code for now
									Value: "/path/to/your/cert.pem",
								},
								{
									Name:  "SOLACE_PRIVATE_KEY", //hard code for now
									Value: "/path/to/your/key.pem",
								},
							},
						},
					},
					ImagePullSecrets: broker.Spec.Monitoring.ImagePullSecrets,
				},
			},
		},
	}
	// Set EventBroker instance as the owner and controller
	ctrl.SetControllerReference(broker, dep, r.Scheme)
	return dep
}

func (r *EventBrokerReconciler) newServiceForPrometheusExporter(exporter *eventbrokerv1alpha1.Monitoring, svcName string, broker *eventbrokerv1alpha1.EventBroker) *corev1.Service {
	dep := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: broker.Namespace,
			Labels:    getBrokerPodSelector(broker.Name, PrometheusExporter),
		},
		Spec: corev1.ServiceSpec{
			Type: exporter.ServiceType,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       getPrometheusExporterPort(exporter),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(getPrometheusExporterPort(exporter))},
				},
			},
			Selector: getBrokerPodSelector(broker.Name, PrometheusExporter),
		},
	}
	// Set EventBroker instance as the owner and controller
	ctrl.SetControllerReference(broker, dep, r.Scheme)
	return dep
}

func getPrometheusExporterPort(broker *eventbrokerv1alpha1.Monitoring) int32 {
	if broker.ContainerPort == 0 {
		return 9628
	}
	return broker.ContainerPort
}

func isSSLVerify(broker *eventbrokerv1alpha1.Monitoring) string {
	if broker.SSLVerify {
		return "https"
	}
	return "http"
}