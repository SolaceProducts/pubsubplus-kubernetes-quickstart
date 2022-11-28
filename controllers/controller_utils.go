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
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"hash/crc64"
	"strconv"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"k8s.io/apimachinery/pkg/types"
)

// Returns the broker pod in the specified role
func (r *PubSubPlusEventBrokerReconciler) getBrokerPod(ctx context.Context, m *eventbrokerv1alpha1.PubSubPlusEventBroker, brokerRole BrokerRole) (*corev1.Pod, error) {
	// List the pods for this pubsubpluseventbroker
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(m.Namespace),
		client.MatchingLabels(getBrokerPodSelector(m.Name, brokerRole)),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		return nil, err
	}
	if podList != nil && len(podList.Items) == 1 {
		return &podList.Items[0], nil
	}
	return nil, fmt.Errorf("filtered broker pod list for broker role %d didn't return exactly one pod", brokerRole)
}


// Returns the TLS secret spec if it exists
func (r *PubSubPlusEventBrokerReconciler) getTlsSecretResourceVersion(ctx context.Context, m *eventbrokerv1alpha1.PubSubPlusEventBroker) (string) {
	var tlsSecretVersion string = ""
    if m.Spec.BrokerTLS.ServerTLsConfigSecret != "" {
        secretName := m.Spec.BrokerTLS.ServerTLsConfigSecret
        foundSecret := &corev1.Secret{}
		// TODO: fix error handling properly
        r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: m.Namespace}, foundSecret)
        // err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: m.Namespace}, foundSecret)
        // if err != nil {
        //     // If a secret name is provided, then it must exist
        //     // You will likely want to create an Event for the user to understand why their reconcile is failing.
        //     return ctrl.Result{}, err
        // }
        tlsSecretVersion = foundSecret.ResourceVersion
    }
	return tlsSecretVersion
}



func brokerSpecHash(s eventbrokerv1alpha1.EventBrokerSpec) string {
	brokerSpecSubset := s.DeepCopy()
	brokerSpecSubset.Monitoring = nil
	return hash(brokerSpecSubset)
}

func convertToByteArray(e any) []byte {
	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.
	err := enc.Encode(e)
	if err != nil {
		return nil
	}
	return network.Bytes()
}

func hash(s any) string {
	crc64Table := crc64.MakeTable(crc64.ECMA)
	return strconv.FormatUint(crc64.Checksum(convertToByteArray(s), crc64Table), 16)
}
