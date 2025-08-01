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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Service test", func() {

	const (
		brokername_service_nonha             = "service-nonha"
		brokername_service_ha                = "service-ha"
		brokername_service_ha_default_config = "service-ha-default-config"
		pvcclaimname                         = "mock-pvc-claim"
		namespace                            = "default"
	)

	var DefaultPort = []*pubsubplus.BrokerPort{
		{
			Name:          "tcp-ssh",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 2222,
			ServicePort:   2222,
		},
	}

	Context("When cluster is created, Service is created", func() {

		It("allows service to be created", func() {

			By("confirming set up works when in NON HA mode", func() {

				var brokerNonHA = &pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_service_nonha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     false,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						Service: pubsubplus.Service{
							ServiceType: corev1.ServiceTypeClusterIP,
							Annotations: map[string]string{
								"Sample": "Test",
							},
							Ports: DefaultPort,
						},
						SecurityContext: pubsubplus.SecurityContext{
							RunAsUser: 0,
							FSGroup:   0,
						},
					},
				}
				Expect(k8sClient.Create(ctx, brokerNonHA)).Should(Succeed())

				//confirm service is created
				EventuallyWithOffset(10, func() bool {
					service := &corev1.Service{}
					serviceName := getObjectName("BrokerService", brokerNonHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: brokerNonHA.Namespace}, service)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker
				Expect(k8sClient.Delete(ctx, brokerNonHA)).To(Succeed())

			})

			By("confirming set up works when in HA mode", func() {

				brokerPVC := &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "data",
						Namespace: namespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resource.MustParse("5Gi"),
							},
						},
					},
				}

				Expect(k8sClient.Create(ctx, brokerPVC)).Should(Succeed())

				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_service_ha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     true,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						Service: pubsubplus.Service{
							ServiceType: corev1.ServiceTypeClusterIP,
							Annotations: map[string]string{
								"Sample": "Test",
							},
							Ports: DefaultPort,
						},
						Storage: pubsubplus.Storage{
							MessagingNodeStorageSize: "0",
							MonitorNodeStorageSize:   "0",
							CustomVolumeMount: []pubsubplus.StorageCustomVolumeMount{
								{
									Name: "Primary",
									PersistentVolumeClaim: pubsubplus.BrokerPersistentVolumeClaim{
										ClaimName: brokerPVC.Name,
									},
								},
							},
						},
						SecurityContext: pubsubplus.SecurityContext{
							RunAsUser: 0,
							FSGroup:   0,
						},
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//service created successfully and can be found
				EventuallyWithOffset(10, func() bool {
					service := &corev1.Service{}
					serviceName := getObjectName("BrokerService", brokerHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: brokerHA.Namespace}, service)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())
				Expect(k8sClient.Delete(ctx, brokerPVC)).To(Succeed())

			})

			By("confirming set up works when in HA mode with default configurations", func() {
				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_service_ha_default_config,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     true,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
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

				//service created successfully
				EventuallyWithOffset(10, func() bool {
					service := &corev1.Service{}
					serviceName := getObjectName("BrokerService", brokerHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: brokerHA.Namespace}, service)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())

			})

			By("confirming NodePort values are set correctly when service type is NodePort", func() {
				brokerNodePort := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service-nodeport",
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     false,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						Service: pubsubplus.Service{
							ServiceType: corev1.ServiceTypeNodePort,
							Annotations: map[string]string{
								"Sample": "Test",
							},
							Ports: []*pubsubplus.BrokerPort{
								{
									Name:          "tcp-semp",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8080,
									ServicePort:   8080,
									NodePort:      30080,
								},
								{
									Name:          "tcp-smf",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 55555,
									ServicePort:   55555,
									NodePort:      30555,
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, &brokerNodePort)).Should(Succeed())

				// Service created successfully
				EventuallyWithOffset(10, func() bool {
					service := &corev1.Service{}
					serviceName := getObjectName("BrokerService", brokerNodePort.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: brokerNodePort.Namespace}, service)
					if err != nil {
						return false
					}

					// Verify service type is NodePort
					if service.Spec.Type != corev1.ServiceTypeNodePort {
						return false
					}

					// Verify nodePort values are set correctly
					for _, port := range service.Spec.Ports {
						if port.Name == "tcp-semp" && port.NodePort != 30080 {
							return false
						}
						if port.Name == "tcp-smf" && port.NodePort != 30555 {
							return false
						}
					}

					return true
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				// Delete broker
				Expect(k8sClient.Delete(ctx, &brokerNodePort)).To(Succeed())
			})

		})
	})

})
