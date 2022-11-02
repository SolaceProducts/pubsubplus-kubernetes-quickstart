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
	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *EventBrokerReconciler) newPodDisruptionBudgetForHADeployment(name string, broker *eventbrokerv1alpha1.EventBroker) *policyv1.PodDisruptionBudget {
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: broker.Namespace,
			Labels:    getBrokerPodSelector(broker.Name, PodDisruptionBudgetHA),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(3)},
			Selector: &metav1.LabelSelector{
				MatchLabels: getBrokerPodSelector(broker.Name, PodDisruptionBudgetHA),
			},
		},
	}
	return pdb
}

// convertToPodDisruptionBudgetBeta1 converts policyv1 version of the PodDisruptionBudget resource to v1beta1
func convertToPodDisruptionBudgetBeta1(toConvert *policyv1.PodDisruptionBudget) *v1beta1.PodDisruptionBudget {
	v1beta1 := &v1beta1.PodDisruptionBudget{}
	v1beta1.ObjectMeta = toConvert.ObjectMeta
	v1beta1.Spec.MinAvailable = toConvert.Spec.MinAvailable
	v1beta1.Spec.Selector = toConvert.Spec.Selector
	v1beta1.Spec.MaxUnavailable = toConvert.Spec.MaxUnavailable
	return v1beta1
}
