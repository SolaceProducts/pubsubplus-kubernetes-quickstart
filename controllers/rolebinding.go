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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	eventbrokerv1beta1 "github.com/SolaceProducts/pubsubplus-operator/api/v1beta1"
)

// roleBindingForEventBroker returns an pubsubpluseventbroker RoleBinding object
func (r *PubSubPlusEventBrokerReconciler) roleBindingForEventBroker(rbName string, m *eventbrokerv1beta1.PubSubPlusEventBroker, sa *corev1.ServiceAccount) *rbacv1.RoleBinding {
	dep := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: sa.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     getObjectName("Role", m.Name),
		},
	}

	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}
