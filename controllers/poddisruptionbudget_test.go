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
	pubsubplus "github.com/SolaceProducts/pubsubplus-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Pod Disruption Budget test", func() {

	const (
		brokername_ha     = "pdb-ha-test"
		brokername_ha_two = "pdb-ha-test-two"
		brokername_nonha  = "pdb-nonha-test"
		namespace         = "default"
	)

	Context("When cluster is created, Pod Disruption Budget is only available in HA", func() {

		It("allows pdb to not be created", func() {

			By("confirming it does not set up in Non HA mode even when PDB is True", func() {

				brokerNonHA := &pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_nonha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:                true,
						Redundancy:               false,
						PodDisruptionBudgetForHA: true,
						UpdateStrategy:           pubsubplus.AutomatedRollingUpdateStrategy,
					},
				}
				Expect(k8sClient.Create(ctx, brokerNonHA)).Should(Succeed())

				//confirm PDB is NOT found
				EventuallyWithOffset(10, func() bool {
					pdb := &policyv1.PodDisruptionBudget{}
					cmName := getObjectName("PodDisruptionBudget", brokerNonHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: brokerNonHA.Namespace}, pdb)
					return err != nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker
				Expect(k8sClient.Delete(ctx, brokerNonHA)).To(Succeed())

			})

			By("confirming it does set up in HA mode when PDB is True", func() {
				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_ha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:                true,
						Redundancy:               true,
						PodDisruptionBudgetForHA: true,
						UpdateStrategy:           pubsubplus.AutomatedRollingUpdateStrategy,
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//PDB created successfully and can be found
				EventuallyWithOffset(10, func() bool {
					pdb := &policyv1.PodDisruptionBudget{}
					cmName := getObjectName("PodDisruptionBudget", brokerHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: brokerHA.Namespace}, pdb)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())

			})

			By("does not set up when in HA mode when PDB is False", func() {
				brokerHATwo := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_ha_two,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:                true,
						Redundancy:               true,
						PodDisruptionBudgetForHA: false,
						UpdateStrategy:           pubsubplus.AutomatedRollingUpdateStrategy,
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHATwo)).Should(Succeed())

				//PDB not Created
				EventuallyWithOffset(10, func() bool {
					pdb := &policyv1.PodDisruptionBudget{}
					cmName := getObjectName("PodDisruptionBudget", brokerHATwo.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: brokerHATwo.Namespace}, pdb)
					return err != nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, &brokerHATwo)).To(Succeed())

			})

		})
	})

})
