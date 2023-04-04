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
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Statefulset test", func() {

	const (
		broker_nonha                = "s-test-nonha"
		broker_ha                   = "s-test-ha"
		broker_ha_prod_level_config = "s-test-ha-prod"
		broker_sample_secret        = "broker-sample-secret"
		preshared_sample_secret     = "preshared-sample-secret"
		admin_sample_secret         = "admin-sample-secret"
		sample_secret               = "sample-secret"
		sample_config               = "sample-config"
		namespace                   = "default"
	)

	Context("When cluster is created, Statefulset is created", func() {

		It("allows statefulset to be created", func() {

			By("confirming set up when in non HA mode primary statefulset can be found", func() {
				tlsSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      sample_secret,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("dummy"),
						"tls.key": []byte("dummy"),
					},
					Type: corev1.SecretTypeTLS,
				}
				Expect(k8sClient.Create(ctx, tlsSecret)).Should(Succeed())
				tlsConfigMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      sample_config,
						Namespace: namespace,
					},
					Data: map[string]string{
						"tls.crt": "dummy",
					},
				}
				Expect(k8sClient.Create(ctx, tlsConfigMap)).Should(Succeed())
				tlsSecretBroker := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      broker_sample_secret,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("dummy"),
						"tls.key": []byte("dummy"),
					},
					Type: corev1.SecretTypeTLS,
				}
				Expect(k8sClient.Create(ctx, tlsSecretBroker)).Should(Succeed())
				presharedSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      preshared_sample_secret,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"preshared_auth_key": []byte("dummypresharedsecretbroker"),
					},
					Type: corev1.SecretTypeOpaque,
				}
				Expect(k8sClient.Create(ctx, presharedSecret)).Should(Succeed())
				adminSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      admin_sample_secret,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						secretKeyName: []byte("dummy-secret-broker"),
					},
					Type: corev1.SecretTypeOpaque,
				}
				Expect(k8sClient.Create(ctx, adminSecret)).Should(Succeed())

				brokerNonHA := &pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      broker_nonha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer: false,
						SystemScaling: &pubsubplus.SystemScaling{
							MaxConnections:      100,
							MaxQueueMessages:    100,
							MaxSpoolUsage:       1000,
							MessagingNodeCpu:    "2",
							MessagingNodeMemory: "2000Mi",
						},
						Redundancy:             true,
						UpdateStrategy:         pubsubplus.AutomatedRollingUpdateStrategy,
						ExtraEnvVarsSecret:     sample_secret,
						ExtraEnvVarsCM:         sample_config,
						PreSharedAuthKeySecret: presharedSecret.Name,
						ExtraEnvVars: []*pubsubplus.ExtraEnvVar{
							{
								Name:  "Sample",
								Value: "Testing",
							},
						},
						PodLabels: map[string]string{
							"Test": "True",
						},
						PodAnnotations: map[string]string{
							"Test": "True",
						},
						BrokerTLS: pubsubplus.BrokerTLS{
							Enabled:               true,
							ServerTLsConfigSecret: tlsSecretBroker.Name,
							TLSCertKeyName:        "tls.crt",
							TLSCertName:           "tls.key",
						},
						Service: pubsubplus.Service{
							ServiceType: corev1.ServiceTypeClusterIP,
							Annotations: map[string]string{
								"Sample": "Test",
							},
							Ports: []*pubsubplus.BrokerPort{},
						},
					},
				}
				Expect(k8sClient.Create(ctx, brokerNonHA)).Should(Succeed())

				//confirm we can find statefulset for primary broker node
				EventuallyWithOffset(10, func() bool {
					statefulset := &v1.StatefulSet{}
					statefulsetName := getStatefulsetName(brokerNonHA.Name, "p")
					err := k8sClient.Get(ctx, types.NamespacedName{Name: statefulsetName, Namespace: brokerNonHA.Namespace}, statefulset)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, brokerNonHA)).To(Succeed())
				Expect(k8sClient.Delete(ctx, tlsConfigMap)).To(Succeed())
				Expect(k8sClient.Delete(ctx, tlsSecret)).To(Succeed())
				Expect(k8sClient.Delete(ctx, tlsSecretBroker)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, adminSecret)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, presharedSecret)).Should(Succeed())

			})

			By("setting it up when in HA mode", func() {
				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      broker_ha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:  true,
						Redundancy: true,
						Timezone:   "UTC",
						ExtraEnvVars: []*pubsubplus.ExtraEnvVar{
							{
								Name:  "Sample",
								Value: "Testing",
							},
						},
						PodLabels: map[string]string{
							"Test": "True",
						},
						PodAnnotations: map[string]string{
							"Test": "True",
						},
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						Storage: pubsubplus.Storage{
							Slow: true,
						},
						Service: pubsubplus.Service{
							ServiceType: corev1.ServiceTypeClusterIP,
							Annotations: map[string]string{
								"Sample": "Test",
							},
							Ports: []*pubsubplus.BrokerPort{},
						},
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//primary statefulset created successfully and can be found in HA mode
				EventuallyWithOffset(10, func() bool {
					statefulset := &v1.StatefulSet{}
					statefulsetName := getStatefulsetName(brokerHA.Name, "p")
					err := k8sClient.Get(ctx, types.NamespacedName{Name: statefulsetName, Namespace: brokerHA.Namespace}, statefulset)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//backup statefulset created successfully and can be found in HA mode
				EventuallyWithOffset(10, func() bool {
					statefulset := &v1.StatefulSet{}
					statefulsetName := getStatefulsetName(brokerHA.Name, "b")
					err := k8sClient.Get(ctx, types.NamespacedName{Name: statefulsetName, Namespace: brokerHA.Namespace}, statefulset)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//monitor statefulset created successfully and can be found in HA mode
				EventuallyWithOffset(10, func() bool {
					statefulset := &v1.StatefulSet{}
					statefulsetName := getStatefulsetName(brokerHA.Name, "m")
					err := k8sClient.Get(ctx, types.NamespacedName{Name: statefulsetName, Namespace: brokerHA.Namespace}, statefulset)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())

			})

			By("set up when in prod-level HA mode", func() {
				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      broker_ha_prod_level_config,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer: false,
						SystemScaling: &pubsubplus.SystemScaling{
							MaxConnections:      100,
							MaxQueueMessages:    100,
							MaxSpoolUsage:       1000,
							MessagingNodeCpu:    "2",
							MessagingNodeMemory: "2000Mi",
						},
						ExtraEnvVarsSecret: "",
						ExtraEnvVarsCM:     "",
						Redundancy:         true,
						Timezone:           "UTC",
						ExtraEnvVars: []*pubsubplus.ExtraEnvVar{
							{
								Name:  "Sample",
								Value: "Testing",
							},
						},
						PodLabels: map[string]string{
							"Test": "True",
						},
						PodAnnotations: map[string]string{
							"Test": "True",
						},
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						Storage: pubsubplus.Storage{
							Slow:                     true,
							MessagingNodeStorageSize: "0",
							MonitorNodeStorageSize:   "0",
						},
						BrokerTLS: pubsubplus.BrokerTLS{
							Enabled: true,
						},
						Service: pubsubplus.Service{
							ServiceType: corev1.ServiceTypeClusterIP,
							Annotations: map[string]string{
								"Sample": "Test",
							},
							Ports: []*pubsubplus.BrokerPort{},
						},
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//primary statefulset created successfully
				EventuallyWithOffset(10, func() bool {
					statefulset := &v1.StatefulSet{}
					statefulsetName := getStatefulsetName(brokerHA.Name, "p")
					err := k8sClient.Get(ctx, types.NamespacedName{Name: statefulsetName, Namespace: brokerHA.Namespace}, statefulset)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//backup statefulset created successfully
				EventuallyWithOffset(10, func() bool {
					statefulset := &v1.StatefulSet{}
					statefulsetName := getStatefulsetName(brokerHA.Name, "b")
					err := k8sClient.Get(ctx, types.NamespacedName{Name: statefulsetName, Namespace: brokerHA.Namespace}, statefulset)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//monitor statefulset created successfully
				EventuallyWithOffset(10, func() bool {
					statefulset := &v1.StatefulSet{}
					statefulsetName := getStatefulsetName(brokerHA.Name, "m")
					err := k8sClient.Get(ctx, types.NamespacedName{Name: statefulsetName, Namespace: brokerHA.Namespace}, statefulset)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())

			})

		})
	})

})
