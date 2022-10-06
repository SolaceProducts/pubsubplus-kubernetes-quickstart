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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
)

func (r *EventBrokerReconciler) serviceForEventBroker(svcName string, m *eventbrokerv1alpha1.EventBroker) *corev1.Service {
	dep := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Name:       "tcp-semp",
					Protocol:   corev1.ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(8080)},
				},
				{
					Name:       "tcp-web",
					Protocol:   corev1.ProtocolTCP,
					Port:       8008,
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(8008)},
				},
				{
					Name:       "tcp-smf",
					Protocol:   corev1.ProtocolTCP,
					Port:       55555,
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(55555)},
				},
			},
			Selector: getServiceSelector(m.Name),
		},
	}
	// Set EventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}
