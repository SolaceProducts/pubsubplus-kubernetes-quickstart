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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
)

// roleForEventBroker returns an eventbroker Role object
func (r *EventBrokerReconciler) roleForEventBroker(m *eventbrokerv1alpha1.EventBroker) *rbacv1.Role {
	roleName := m.Name + "-pubsubplus-podtagupdater"

	dep := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   m.Name,
				"app.kubernetes.io/name":       "eventbroker",
				"app.kubernetes.io/managed-by": "solace-pubsubplus-operator",
			},
		},
		Rules:      []rbacv1.PolicyRule{
			{
				Verbs:           []string{
					"patch",
				},
				APIGroups:       []string{
					"",
				},
				Resources:       []string{
					"pods",
				},
			},
		},
	}

	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

