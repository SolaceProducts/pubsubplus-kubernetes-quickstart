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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	eventbrokerv1beta1 "github.com/SolaceProducts/pubsubplus-operator/api/v1beta1"
)

// roleForEventBroker returns an pubsubpluseventbroker Role object
func (r *PubSubPlusEventBrokerReconciler) roleForEventBroker(roleName string, m *eventbrokerv1beta1.PubSubPlusEventBroker) *rbacv1.Role {
	dep := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"patch",
				},
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods",
				},
			},
		},
	}

	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}
