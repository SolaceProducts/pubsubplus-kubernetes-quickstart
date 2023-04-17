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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Configmap test", func() {

	const (
		brokername_configmap_nonha = "configmap-nonha"
		brokername_configmap_ha    = "configmap-ha"
		namespace                  = "default"
	)

	Context("When cluster is created, Configmap is created", func() {

		It("allows configmap to be created and deleted", func() {

			By("setting it up when in Non HA mode", func() {

				brokerNonHA := &pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_configmap_nonha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     false,
						Timezone:       "UTC",
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
					},
				}
				Expect(k8sClient.Create(ctx, brokerNonHA)).Should(Succeed())

				//
				EventuallyWithOffset(10, func() bool {
					cm := &corev1.ConfigMap{}
					cmName := getObjectName("ConfigMap", brokerNonHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: brokerNonHA.Namespace}, cm)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up configmap
				Expect(k8sClient.Delete(ctx, brokerNonHA)).To(Succeed())

			})

			By("setting it up when in HA mode", func() {
				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_configmap_ha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     true,
						Timezone:       "UTC",
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//config map created successfully
				EventuallyWithOffset(10, func() bool {
					cm := &corev1.ConfigMap{}
					cmName := getObjectName("ConfigMap", brokerHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: brokerHA.Namespace}, cm)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up configmap
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())

			})

		})
	})

})
