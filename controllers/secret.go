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
	"crypto/rand"
	"math/big"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	eventbrokerv1beta1 "github.com/SolaceProducts/pubsubplus-operator/api/v1beta1"
)

// secretForEventBroker returns an pubsubpluseventbroker Secret object
func (r *PubSubPlusEventBrokerReconciler) secretForEventBroker(adminSecretName string, m *eventbrokerv1beta1.PubSubPlusEventBroker) *corev1.Secret {

	randomPassword := generateSimplePassword(10)

	dep := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminSecretName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Data: map[string][]byte{
			secretKeyName: []byte(randomPassword),
		},
		Type: corev1.SecretTypeOpaque,
	}

	// FOR NOW. May be reconsidered.
	// Set PubSubPlusEventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// createPreSharedAuthKeySecret returns an PubSubPlusEventBroker PreSharedAuthKeySecret object
func (r *PubSubPlusEventBrokerReconciler) createPreSharedAuthKeySecret(preSharedAuthKeySecretName string, m *eventbrokerv1beta1.PubSubPlusEventBroker) *corev1.Secret {

	randomPassword := generateSimplePassword(50)

	dep := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      preSharedAuthKeySecretName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Data: map[string][]byte{
			preSharedAuthKeyName: []byte(randomPassword),
		},
		Type: corev1.SecretTypeOpaque,
	}
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// monitoringSecretForEventBroker returns a Secret object to be used by Exporter
func (r *PubSubPlusEventBrokerReconciler) monitoringSecretForEventBroker(monitoringSecretName string, m *eventbrokerv1beta1.PubSubPlusEventBroker) *corev1.Secret {

	randomPassword := generateSimplePassword(10)

	dep := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      monitoringSecretName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Data: map[string][]byte{
			monitorSecretKeyName: []byte(randomPassword),
		},
		Type: corev1.SecretTypeOpaque,
	}
	// Set PubSubPlusEventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

func generateSimplePassword(length int) string {
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b.WriteRune(chars[nBig.Int64()])
	}
	return b.String()
}
