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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("RoleBinding test for operator", func() {

	const (
		brokername_role_nonha = "rolebinding-nonha"
		brokername_role_ha    = "rolebinding-ha"
		namespace             = "default"
	)

	Context("When cluster is created, RoleBinding is created", func() {

		It("allows rolebinding to be created", func() {

			By("setting it up when in Non HA mode", func() {

				brokerNonHA := &pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_role_nonha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     false,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
					},
				}
				Expect(k8sClient.Create(ctx, brokerNonHA)).Should(Succeed())

				//roleBinding created successfully and can be found
				EventuallyWithOffset(10, func() bool {
					roleBinding := &rbacv1.RoleBinding{}
					roleBindingName := getObjectName("RoleBinding", brokerNonHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: roleBindingName, Namespace: brokerNonHA.Namespace}, roleBinding)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, brokerNonHA)).To(Succeed())

			})

			By("setting it up when in non HA mode", func() {
				brokerHA := pubsubplus.PubSubPlusEventBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      brokername_role_ha,
						Namespace: namespace,
					},
					Spec: pubsubplus.EventBrokerSpec{
						Developer:      true,
						Redundancy:     true,
						UpdateStrategy: pubsubplus.AutomatedRollingUpdateStrategy,
						SecurityContext: pubsubplus.SecurityContext{
							RunAsUser: 0,
							FSGroup:   0,
						},
					},
				}
				Expect(k8sClient.Create(ctx, &brokerHA)).Should(Succeed())

				//roleBinding created successfully and can be found
				EventuallyWithOffset(10, func() bool {
					roleBinding := &rbacv1.RoleBinding{}
					roleBindingName := getObjectName("RoleBinding", brokerHA.Name)
					err := k8sClient.Get(ctx, types.NamespacedName{Name: roleBindingName, Namespace: brokerHA.Namespace}, roleBinding)
					return err == nil
				}).WithTimeout(20 * time.Second).Should(BeTrue())

				//delete broker and clean up
				Expect(k8sClient.Delete(ctx, &brokerHA)).To(Succeed())

			})

		})
	})

})
