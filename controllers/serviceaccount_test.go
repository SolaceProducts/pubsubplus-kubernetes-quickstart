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

var _ = Describe("ServiceAccount Test for Operator", func() {

	const (
		brokername_serviceaccount_nonha = "serviceacccount-nonha"
		brokername_serviceaccount_ha    = "serviceacccount-ha"
		namespace                       = "default"
	)

	Context("When cluster is created, ServiceAccount is created", func() {

		It("allows serviceAccount to be created", func() {

			By("setting it up when in NON HA mode", func() {

				brokerNonHA := &pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_serviceaccount_nonha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     false,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						BrokerNodeAssignment: []pubsubplus.NodeAssignment{
							{
								Name: "Primary",
								Spec: pubsubplus.NodeAssignmentSpec{
									NodeSelector: map[string]string{
										"kubernetes.io/os": "linux",
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, brokerNonHA)).Should(Succeed())

				//serviceAccount created successfully and can be found
				EventuallyWithOffset(10, func() bool {
					serviceAccount := &corev1.ServiceAccount{}
					serviceAccountName := getObjectName("ServiceAccount", brokerNonHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: brokerNonHA.Namespace}, serviceAccount)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker
				Expect(k8sClient.Delete(ctx, brokerNonHA)).To(Succeed())

			})

			By("confirming it is set up when in HA mode", func() {
				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_serviceaccount_ha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     true,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						BrokerNodeAssignment: []pubsubplus.NodeAssignment{
							{
								Name: "Primary",
								Spec: pubsubplus.NodeAssignmentSpec{
									NodeSelector: map[string]string{
										"kubernetes.io/os": "linux",
									},
								},
							}, {
								Name: "Backup",
								Spec: pubsubplus.NodeAssignmentSpec{
									NodeSelector: map[string]string{
										"kubernetes.io/os": "linux",
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//serviceAccount created successfully and can be found
				EventuallyWithOffset(10, func() bool {
					serviceAccount := &corev1.ServiceAccount{}
					serviceAccountName := getObjectName("ServiceAccount", brokerHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: brokerHA.Namespace}, serviceAccount)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())

			})

		})
	})
})
