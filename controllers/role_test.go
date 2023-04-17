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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Role test for Exporter", func() {

	const (
		brokername_role_nonha = "role-nonha"
		brokername_role_ha    = "role-ha"
		namespace             = "default"
	)

	Context("When cluster is created, Role is created", func() {

		It("allows role to be created", func() {

			By("confirming Role is created in Non HA mode", func() {
				brokerNonHA := &pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_role_nonha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     false,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						Monitoring: pubsubplus.Monitoring{
							Enabled: false,
							MonitoringImage: &pubsubplus.MonitoringImage{
								ImagePullPolicy:  corev1.PullIfNotPresent,
								ImagePullSecrets: []corev1.LocalObjectReference{},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, brokerNonHA)).Should(Succeed())

				//confirm role is created and found
				EventuallyWithOffset(10, func() bool {
					role := &rbacv1.Role{}
					roleName := getObjectName("Role", brokerNonHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: roleName, Namespace: brokerNonHA.Namespace}, role)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, brokerNonHA)).To(Succeed())

			})

			By("confirming Role is created in HA mode", func() {
				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_role_ha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     true,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//role created successfully
				EventuallyWithOffset(10, func() bool {
					role := &rbacv1.Role{}
					roleName := getObjectName("Role", brokerHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: roleName, Namespace: brokerHA.Namespace}, role)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())

			})

		})
	})

})
