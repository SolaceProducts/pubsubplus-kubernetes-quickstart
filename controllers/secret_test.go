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

var _ = Describe("Testing Secret for Operator", func() {

	const (
		brokername_secret_nonha = "secret-nonha"
		brokername_secret_ha    = "secret-ha"
		tls_secret              = "secret-s-tls"
		namespace               = "default"
	)

	Context("When cluster is created, Secret is created", func() {

		It("allows secret to be created", func() {

			By("setting it up when in Non HA mode", func() {

				tlsSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tls_secret,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("dummy"),
						"tls.key": []byte("dummy"),
					},
					Type: corev1.SecretTypeTLS,
				}
				Expect(k8sClient.Create(ctx, tlsSecret)).Should(Succeed())

				var brokerNonHA = &pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_secret_nonha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     false,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						BrokerTLS: pubsubplus.BrokerTLS{
							Enabled:               true,
							ServerTLsConfigSecret: tlsSecret.Name,
							TLSCertName:           "tls.crt",
							TLSCertKeyName:        "tls.key",
						},
					},
				}
				Expect(k8sClient.Create(ctx, brokerNonHA)).Should(Succeed())

				//confirm that secret can be found
				EventuallyWithOffset(10, func() bool {
					secret := &corev1.Secret{}
					secretName := getObjectName("AdminCredentialsSecret", brokerNonHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: brokerNonHA.Namespace}, secret)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, brokerNonHA)).To(Succeed())
				Expect(k8sClient.Delete(ctx, tlsSecret)).To(Succeed())

			})

			By("setting it up when in HA mode", func() {
				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_secret_ha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     true,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//secret created successfully and can be found
				EventuallyWithOffset(10, func() bool {
					secret := &corev1.Secret{}
					secretName := getObjectName("AdminCredentialsSecret", brokerHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: brokerHA.Namespace}, secret)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())

			})

		})
	})

})
