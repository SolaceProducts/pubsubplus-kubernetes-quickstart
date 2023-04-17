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

var _ = Describe("Discovery Service Test for Operator", func() {

	const (
		broker_nonha      = "dservice-test-nonha"
		broker_ha         = "dservice-test-ha"
		monitoring_secret = "monitoring-ds-secret"
		namespace         = "default"
	)

	Context("When cluster is created, DiscoveryService is created", func() {

		It("allows service to be created", func() {

			By("confirming it does set up when in non HA mode", func() {

				var brokerNonHA = &pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      broker_nonha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:          true,
						Redundancy:         false,
						UpdateStrategy:     pubsubplus.AutomatedRollingUpdateStrategy,
						ExtraEnvVarsCM:     "",
						ExtraEnvVarsSecret: "",
					},
				}
				Expect(k8sClient.Create(ctx, brokerNonHA)).Should(Succeed())

				EventuallyWithOffset(10, func() bool {
					service := &corev1.Service{}
					serviceName := getObjectName("DiscoveryService", brokerNonHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: brokerNonHA.Namespace}, service)
					return err != nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker
				Expect(k8sClient.Delete(ctx, brokerNonHA)).To(Succeed())

			})

			By("confirming it does set up when in HA mode", func() {

				monitoringSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      monitoring_secret,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						monitorSecretKeyName: []byte("dummymonitoringsecretbroker"),
					},
					Type: corev1.SecretTypeOpaque,
				}
				Expect(k8sClient.Create(ctx, monitoringSecret)).Should(Succeed())

				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      broker_ha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:                   false,
						Redundancy:                  true,
						UpdateStrategy:              pubsubplus.AutomatedRollingUpdateStrategy,
						MonitoringCredentialsSecret: monitoringSecret.Name,
						Storage: pubsubplus.Storage{
							Slow:                     true,
							MessagingNodeStorageSize: "0",
							MonitorNodeStorageSize:   "0",
						},
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//service created successfully
				EventuallyWithOffset(10, func() bool {
					service := &corev1.Service{}
					serviceName := getObjectName("DiscoveryService", brokerHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: brokerHA.Namespace}, service)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())
				Expect(k8sClient.Delete(ctx, monitoringSecret)).To(Succeed())

			})

		})
	})

})
