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
	"encoding/json"
	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *PubSubPlusEventBrokerReconciler) serviceForEventBroker(svcName string, m *eventbrokerv1alpha1.PubSubPlusEventBroker) *corev1.Service {
	dep := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Spec: corev1.ServiceSpec{
			Type:     getServiceType(m.Spec.Service),
			Selector: getServiceSelector(m.Name),
		},
	}
	if len(m.Spec.Service.Ports) > 0 {
		ports := make([]corev1.ServicePort, len(m.Spec.Service.Ports))
		for idx, pbPort := range m.Spec.Service.Ports {
			ports[idx] = corev1.ServicePort{
				Name:       pbPort.Name,
				Protocol:   pbPort.Protocol,
				Port:       pbPort.ServicePort,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: pbPort.ContainerPort},
			}
		}
		dep.Spec.Ports = ports
	} else {
		portConfig := eventbrokerv1alpha1.Service{}
		json.Unmarshal([]byte(DefaultServiceConfig), &portConfig)
		ports := make([]corev1.ServicePort, len(portConfig.Ports))
		for idx, pbPort := range portConfig.Ports {
			ports[idx] = corev1.ServicePort{
				Name:       pbPort.Name,
				Protocol:   pbPort.Protocol,
				Port:       pbPort.ServicePort,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: pbPort.ContainerPort},
			}
		}
		dep.Spec.Ports = ports
	}

	if m.Spec.Service.Annotations != nil || len(m.Spec.Service.Annotations) > 0 {
		dep.Annotations = m.Spec.Service.Annotations
	}
	// Set PubSubPlusEventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

func getServiceType(ms eventbrokerv1alpha1.Service) corev1.ServiceType {
	if ms.ServiceType != "" {
		return ms.ServiceType
	}
	return corev1.ServiceTypeLoadBalancer
}
