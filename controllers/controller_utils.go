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
	"embed"
	"encoding/gob"
	"fmt"
	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
	"hash/crc64"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

var (
	//go:embed brokerscripts configs
	scripts embed.FS
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

// Returns the TLS secret resourceVersion if it exists. If TLS is not configured or TLS secret not found it returns empty string
func (r *PubSubPlusEventBrokerReconciler) tlsSecretHash(ctx context.Context, m *eventbrokerv1alpha1.PubSubPlusEventBroker) string {
	var tlsSecretVersion string = ""
	if m.Spec.BrokerTLS.ServerTLsConfigSecret != "" {
		secretName := m.Spec.BrokerTLS.ServerTLsConfigSecret
		foundSecret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: m.Namespace}, foundSecret)
		if err == nil {
			tlsSecretVersion = foundSecret.ResourceVersion
		}
	}
	return tlsSecretVersion
}

func brokerSpecHash(s eventbrokerv1alpha1.EventBrokerSpec) string {
	brokerSpecSubset := s.DeepCopy()
	// Mask anything that is not relevant to the StatefulSet / broker Pods
	brokerSpecSubset.Monitoring = eventbrokerv1alpha1.Monitoring{}
	brokerSpecSubset.Service.Annotations = nil
	brokerSpecSubset.Service.ServiceType = corev1.ServiceTypeLoadBalancer        // cannot use nil, setting it a constant value
	brokerSpecSubset.Redundancy = false                                          // change of redundancy is not supported for now
	brokerSpecSubset.ServiceAccount = eventbrokerv1alpha1.BrokerServiceAccount{} // change of SA is not supported
	brokerSpecSubset.AdminCredentialsSecret = ""
	brokerSpecSubset.PreSharedAuthKeySecret = ""
	brokerSpecSubset.PodDisruptionBudgetForHA = false // does not affect the statefulset/pod
	return hash(brokerSpecSubset.String())
}

func brokerServiceHash(s eventbrokerv1alpha1.EventBrokerSpec) string {
	brokerServiceSubset := s.Service.DeepCopy()
	return hash(brokerServiceSubset.String())
}

func brokerServiceOutdated(service *corev1.Service, expectedBrokerServiceHash string) bool {
	result := service.ObjectMeta.Annotations[brokerServiceSignatureAnnotationName] != expectedBrokerServiceHash
	return result
}

func brokerStsOutdated(sts *appsv1.StatefulSet, expectedBrokerSpecHash string, expectedTlsSecretHash string) bool {
	result := sts.Spec.Template.ObjectMeta.Annotations[brokerSpecSignatureAnnotationName] != expectedBrokerSpecHash
	// Ignore expectedTlsSecretHash if it is an empty string. This means the sts is not marked as outdated if the secret does not exist or has been deleted
	if expectedTlsSecretHash != "" {
		result = result || (sts.Spec.Template.ObjectMeta.Annotations[tlsSecretSignatureAnnotationName] != expectedTlsSecretHash)
	}
	return result
}

func brokerPodOutdated(pod *corev1.Pod, expectedBrokerSpecHash string, expectedTlsSecretHash string) bool {
	result := pod.ObjectMeta.Annotations[brokerSpecSignatureAnnotationName] != expectedBrokerSpecHash
	// Ignore expectedTlsSecretHash if it is an empty string. This means the pod is not marked as outdated if the secret does not exist or has been deleted
	if expectedTlsSecretHash != "" {
		result = result || (pod.ObjectMeta.Annotations[tlsSecretSignatureAnnotationName] != expectedTlsSecretHash)
	}
	return result
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
