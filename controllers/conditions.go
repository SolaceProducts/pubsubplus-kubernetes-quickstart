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
	"context"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionName string

const (
	NoWarningsCondition      = "NoWarnings"
	ServiceReadyCondition    = "ServiceReady"
	HAReadyCondition         = "HAReady"
	MonitoringReadyCondition = "MonitoringReady"
)

type ConditionReason string

const (
	ResourceErrorReason                      = "ResourceError"
	MaintenanceModeActiveReason              = "MaintenanceModeActive"
	NoIssuesReason                           = "NoIssues"
	AllBrokersHAReadyInRedundancyGroupReason = "AllBrokersHAReadyInRedundancyGroup"
	MonitoringReadyReason                    = "MonitoringReady"
	WaitingForActivePodReason                = "WaitingForActivePod"
	ActivePodAndServiceExistsReason          = "ActivePodAndServiceExists"
)

// sets or updates a status condition using helper from meta
func (r *PubSubPlusEventBrokerReconciler) SetCondition(ctx context.Context, log logr.Logger, eb *eventbrokerv1alpha1.PubSubPlusEventBroker, condition ConditionName, status metav1.ConditionStatus, reason ConditionReason, message string) error {
	if eb.Status.Conditions == nil {
		eb.Status.Conditions = []metav1.Condition{}
	}
	meta.SetStatusCondition(&eb.Status.Conditions, metav1.Condition{
		Type:    string(condition),
		Status:  status,
		Reason:  string(reason),
		Message: message,
	})
	error := r.Status().Update(ctx, eb)
	if error != nil {
		log.Error(error, "Unable to update status with condition", "Condition", condition)
	}
	return error
}
