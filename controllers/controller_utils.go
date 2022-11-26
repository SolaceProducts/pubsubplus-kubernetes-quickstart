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
)

var DefaultServiceConfig = string(`{"type":"LoadBalancer","annotations":{},"ports":[{"servicePort":2222,"containerPort":2222,"protocol":"TCP","name":"tcp-ssh"},{"servicePort":8080,"containerPort":8080,"protocol":"TCP","name":"tcp-semp"},{"servicePort":1943,"containerPort":1943,"protocol":"TCP","name":"tls-semp"},{"servicePort":55555,"containerPort":55555,"protocol":"TCP","name":"tcp-smf"},{"servicePort":55003,"containerPort":55003,"protocol":"TCP","name":"tcp-smfcomp"},{"servicePort":55443,"containerPort":55443,"protocol":"TCP","name":"tls-smf"},{"servicePort":55556,"containerPort":55556,"protocol":"TCP","name":"tcp-smfroute"},{"servicePort":8008,"containerPort":8008,"protocol":"TCP","name":"tcp-web"},{"servicePort":1443,"containerPort":1443,"protocol":"TCP","name":"tls-web"},{"servicePort":9000,"containerPort":9000,"protocol":"TCP","name":"tcp-rest"},{"servicePort":9443,"containerPort":9443,"protocol":"TCP","name":"tls-rest"},{"servicePort":5672,"containerPort":5672,"protocol":"TCP","name":"tcp-amqp"},{"servicePort":5671,"containerPort":5671,"protocol":"TCP","name":"tls-amqp"},{"servicePort":1883,"containerPort":1883,"protocol":"TCP","name":"tcp-mqtt"},{"servicePort":8883,"containerPort":8883,"protocol":"TCP","name":"tls-mqtt"},{"servicePort":8000,"containerPort":8000,"protocol":"TCP","name":"tcp-mqttweb"},{"servicePort":8443,"containerPort":8443,"protocol":"TCP","name":"tls-mqttweb"}]}`)

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
	return nil, fmt.Errorf("filtered broker pod list didn't return exactly one pod")
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
