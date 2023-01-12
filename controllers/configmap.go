/*
Copyright 2022.

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
	"embed"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
)

var (
	//go:embed brokerscripts
	scripts embed.FS
)

func (r *PubSubPlusEventBrokerReconciler) configmapForEventBroker(cmName string, m *eventbrokerv1alpha1.PubSubPlusEventBroker) *corev1.ConfigMap {

	InitSh, _ := scripts.ReadFile("brokerscripts/init.sh")
	StartupBrokerSh, _ := scripts.ReadFile("brokerscripts/startup-broker.sh")
	ReadinessCheckSh, _ := scripts.ReadFile("brokerscripts/readiness_check.sh")
	SempQuerySh, _ := scripts.ReadFile("brokerscripts/semp_query.sh")

	dep := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Data: map[string]string{
			"init.sh":            string(InitSh),
			"startup-broker.sh":  string(StartupBrokerSh),
			"readiness_check.sh": string(ReadinessCheckSh),
			"semp_query.sh":      string(SempQuerySh),
		},
	}
	// Set PubSubPlusEventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}
