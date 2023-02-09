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
	"fmt"
	"strconv"

	// "reflect"
	"strings"
	"time"

	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/client-go/tools/record"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// PubSubPlusEventBrokerReconciler reconciles a PubSubPlusEventBroker object
type PubSubPlusEventBrokerReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Recorder    record.EventRecorder
	IsOpenShift bool
}

const (
	dependencyTlsSecretField = ".spec.tls.serverTlsConfigSecret"
)

// TODO: review and revise to minimum at the end of the dev cycle!
// e.g.: "controllers are granted read-only access to spec but full write access to status"
//+kubebuilder:rbac:groups=pubsubplus.solace.com,resources=pubsubpluseventbrokers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pubsubplus.solace.com,resources=pubsubpluseventbrokers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pubsubplus.solace.com,resources=pubsubpluseventbrokers/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PubSubPlusEventBrokerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Safeguards in case reconcile gets stuck - not used for now
	// ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	// defer cancel()
	// Format is set in main.go
	log := ctrllog.FromContext(ctx)

	var stsP, stsB, stsM *appsv1.StatefulSet

	// Fetch the PubSubPlusEventBroker instance
	pubsubpluseventbroker := &eventbrokerv1alpha1.PubSubPlusEventBroker{}
	err := r.Get(ctx, req.NamespacedName, pubsubpluseventbroker)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("PubSubPlusEventBroker resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object
		r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to read PubSubPlusEventBroker manifest")
		return ctrl.Result{}, err
	} else {
		log.V(1).Info("Detected existing pubsubpluseventbroker", " pubsubpluseventbroker.Name", pubsubpluseventbroker.Name)
	}

	// Check maintenance mode
	if labelValue, ok := pubsubpluseventbroker.Labels[maintenanceLabel]; ok && labelValue == "true" {
		msg := fmt.Sprintf("Found maintenance label '%s=true', reconcile paused.", maintenanceLabel)
		log.Info(msg)
		r.SetCondition(ctx, log, pubsubpluseventbroker, NoWarningsCondition, metav1.ConditionFalse, MaintenanceModeActiveReason, msg)
		return ctrl.Result{}, nil
	}

	// Check if new ServiceAccount needs to created or an existing one needs to be used
	sa := &corev1.ServiceAccount{}
	if len(strings.TrimSpace(pubsubpluseventbroker.Spec.ServiceAccount.Name)) > 0 {
		err = r.Get(ctx, types.NamespacedName{Name: pubsubpluseventbroker.Spec.ServiceAccount.Name, Namespace: pubsubpluseventbroker.Namespace}, sa)
		if err != nil && errors.IsNotFound(err) {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to find specified ServiceAccount", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", pubsubpluseventbroker.Spec.ServiceAccount.Name)
			return ctrl.Result{}, err
		}
		log.V(1).Info("Found specified ServiceAccount", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", sa.Name)
	} else {
		// Check if ServiceAccount already exists, if not create a new one
		saName := getObjectName("ServiceAccount", pubsubpluseventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: saName, Namespace: pubsubpluseventbroker.Namespace}, sa)
		if err != nil && errors.IsNotFound(err) {
			// Define a new ServiceAccount
			sa := r.serviceAccountForEventBroker(saName, pubsubpluseventbroker)
			log.Info("Creating a new ServiceAccount", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", sa.Name)
			err = r.Create(ctx, sa)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new ServiceAccount", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", sa.Name)
				return ctrl.Result{}, err
			}
			// ServiceAccount created successfully - return and requeue
			r.emitResourceSuccessEvent(pubsubpluseventbroker, "ServiceAccount", saName)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get ServiceAccount")
			return ctrl.Result{}, err
		} else {
			log.V(1).Info("Detected existing ServiceAccount", " ServiceAccount.Name", sa.Name)
		}
	}

	// Check if Podtagupdater Role already exists, if not create a new one
	role := &rbacv1.Role{}
	roleName := getObjectName("Role", pubsubpluseventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: roleName, Namespace: pubsubpluseventbroker.Namespace}, role)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Role
		role := r.roleForEventBroker(roleName, pubsubpluseventbroker)
		log.Info("Creating a new Role", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
		err = r.Create(ctx, role)
		if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Role", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
			return ctrl.Result{}, err
		}
		// Role created successfully - return and requeue
		r.emitResourceSuccessEvent(pubsubpluseventbroker, "Podtagupdater Role", roleName)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Role")
		return ctrl.Result{}, err
	} else {
		log.V(1).Info("Detected existing Role", " Role.Name", role.Name)
	}

	// Check if RoleBinding already exists, if not create a new one
	rb := &rbacv1.RoleBinding{}
	rbName := getObjectName("RoleBinding", pubsubpluseventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: rbName, Namespace: pubsubpluseventbroker.Namespace}, rb)
	if err != nil && errors.IsNotFound(err) {
		// Define a new RoleBinding
		rb := r.roleBindingForEventBroker(rbName, pubsubpluseventbroker, sa)
		log.Info("Creating a new RoleBinding", "RoleBinding.Namespace", rb.Namespace, "RoleBinding.Name", rb.Name)
		err = r.Create(ctx, rb)
		if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new RoleBinding", "RoleBinding.Namespace", rb.Namespace, "RoleBinding.Name", rb.Name)
			return ctrl.Result{}, err
		}
		// RoleBinding created successfully - return and requeue
		r.emitResourceSuccessEvent(pubsubpluseventbroker, "RoleBinding", rbName)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get RoleBinding")
		return ctrl.Result{}, err
	} else {
		log.V(1).Info("Detected existing RoleBinding", " RoleBinding.Name", rb.Name)
	}

	// Check if the ConfigMap already exists, if not create a new one
	cm := &corev1.ConfigMap{}
	cmName := getObjectName("ConfigMap", pubsubpluseventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: pubsubpluseventbroker.Namespace}, cm)
	if err != nil && errors.IsNotFound(err) {
		// Define a new configmap
		cm := r.configmapForEventBroker(cmName, pubsubpluseventbroker)
		log.Info("Creating a new ConfigMap", "Configmap.Namespace", cm.Namespace, "Configmap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new ConfigMap", "Configmap.Namespace", cm.Namespace, "Configmap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		// ConfigMap created successfully - return and requeue
		r.emitResourceSuccessEvent(pubsubpluseventbroker, "ConfigMap", cmName)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	} else {
		log.V(1).Info("Detected existing ConfigMap", " ConfigMap.Name", cm.Name)
	}

	// Check if the Broker Service already exists, if not create a new one
	brokerServiceHash := brokerServiceHash(pubsubpluseventbroker.Spec)
	svc := &corev1.Service{}
	svcName := getObjectName("BrokerService", pubsubpluseventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: pubsubpluseventbroker.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		svc := r.createServiceForEventBroker(svcName, pubsubpluseventbroker)
		log.Info("Creating a new Broker Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Broker Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{}, err
		}
		// Broker Service created successfully - return and requeue
		r.emitResourceSuccessEvent(pubsubpluseventbroker, "Broker Service", svcName)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Broker Service")
		return ctrl.Result{}, err
	} else {
		// Check if existing service is not outdated at this point
		if brokerServiceOutdated(svc, brokerServiceHash) {
			log.Info("Updating existing Broker Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			r.updateServiceForEventBroker(svc, pubsubpluseventbroker)
			err = r.Update(ctx, svc)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to update Broker Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
				return ctrl.Result{}, err
			}
			// Service updated successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.V(1).Info("Detected up-to-date existing Broker Service", " Service.Name", svc.Name)
	}

	haDeployment := pubsubpluseventbroker.Spec.Redundancy
	if haDeployment {
		// Check if the Discovery Service already exists, if not create a new one
		dsvc := &corev1.Service{}
		dsvcName := getObjectName("DiscoveryService", pubsubpluseventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: dsvcName, Namespace: pubsubpluseventbroker.Namespace}, dsvc)
		if err != nil && errors.IsNotFound(err) {
			// Define a new service
			svc := r.discoveryserviceForEventBroker(dsvcName, pubsubpluseventbroker)
			log.Info("Creating a new Discovery Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			err = r.Create(ctx, svc)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Discovery Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
				return ctrl.Result{}, err
			}
			// Discovery Service created successfully - return and requeue
			r.emitResourceSuccessEvent(pubsubpluseventbroker, "Discovery Service", dsvcName)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Discovery Service")
			return ctrl.Result{}, err
		} else {
			log.V(1).Info("Detected existing Discovery Discovery Service", " Service.Name", dsvc.Name)
		}
	}

	// Check Admin Credentials Secret
	adminSecret := &corev1.Secret{}
	if len(strings.TrimSpace(pubsubpluseventbroker.Spec.AdminCredentialsSecret)) == 0 {
		adminSecretName := getObjectName("AdminCredentialsSecret", pubsubpluseventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: pubsubpluseventbroker.Namespace}, adminSecret)
		if err != nil && errors.IsNotFound(err) {
			// Define a new Admin Credentials Secret
			secret := r.secretForEventBroker(adminSecretName, pubsubpluseventbroker)
			log.Info("Creating a new Admin Credentials Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
			err = r.Create(ctx, secret)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Admin Credentials Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
				return ctrl.Result{}, err
			}
			// Admin Credentials Secret created successfully - return and requeue
			r.emitResourceSuccessEvent(pubsubpluseventbroker, "Admin Credentials Secret", adminSecretName)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Admin Credentials Secret")
			return ctrl.Result{}, err
		} else {
			log.V(1).Info("Detected existing Admin Credentials Secret", " Secret.Name", adminSecret.Name)
		}
	} else {
		err = r.Get(ctx, types.NamespacedName{Name: pubsubpluseventbroker.Spec.AdminCredentialsSecret, Namespace: pubsubpluseventbroker.Namespace}, adminSecret)
		if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to find specified Admin Credentials Secret: '"+pubsubpluseventbroker.Spec.AdminCredentialsSecret+"'")
			return ctrl.Result{}, err
		} else {
			log.V(1).Info("Detected specified Admin Credentials Secret", " Secret.Name", adminSecret.Name)
		}
	}

	monitoringSecret := &corev1.Secret{}
	// Check Monitoring Exporter Secret
	if len(strings.TrimSpace(pubsubpluseventbroker.Spec.MonitoringCredentialsSecret)) == 0 {
		monitoringSecretName := getObjectName("MonitoringCredentialsSecret", pubsubpluseventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: monitoringSecretName, Namespace: pubsubpluseventbroker.Namespace}, monitoringSecret)
		if err != nil && errors.IsNotFound(err) {
			// Define a new Monitoring Exporter Secret
			monitoringSecret = r.monitoringSecretForEventBroker(monitoringSecretName, pubsubpluseventbroker)
			log.Info("Creating a new Monitoring Exporter Secret", "Secret.Namespace", monitoringSecret.Namespace, "Secret.Name", monitoringSecret.Name)
			err = r.Create(ctx, monitoringSecret)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Monitoring Exporter Secret", "Secret.Namespace", monitoringSecret.Namespace, "Secret.Name", monitoringSecret.Name)
				return ctrl.Result{}, err
			}
			// Monitoring Exporter Secret created successfully - return and requeue
			r.emitResourceSuccessEvent(pubsubpluseventbroker, "Monitoring Exporter Secret", monitoringSecretName)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Monitoring Exporter Secret")
			return ctrl.Result{}, err
		} else {
			log.V(1).Info("Detected existing Monitoring Exporter Secret", " Secret.Name", monitoringSecret.Name)
		}
	} else {
		err = r.Get(ctx, types.NamespacedName{Name: pubsubpluseventbroker.Spec.MonitoringCredentialsSecret, Namespace: pubsubpluseventbroker.Namespace}, monitoringSecret)
		if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to find specified Monitoring Exporter Secret: '"+pubsubpluseventbroker.Spec.MonitoringCredentialsSecret+"'")
			return ctrl.Result{}, err
		} else {
			log.V(1).Info("Detected specified Monitoring Exporter Secret", " Secret.Name", monitoringSecret.Name)
		}
	}

	preSharedAuthKeySecret := &corev1.Secret{}
	//preSharedAuthKeySecret is only used in HA mode
	if haDeployment {
		if len(strings.TrimSpace(pubsubpluseventbroker.Spec.PreSharedAuthKeySecret)) == 0 {
			preSharedAuthSecretName := getObjectName("PreSharedAuthSecret", pubsubpluseventbroker.Name)
			err = r.Get(ctx, types.NamespacedName{Name: preSharedAuthSecretName, Namespace: pubsubpluseventbroker.Namespace}, preSharedAuthKeySecret)
			if err != nil && errors.IsNotFound(err) {
				// Define a new PreShareAuthSecret
				preSharedAuthKeySecret := r.createPreSharedAuthKeySecret(preSharedAuthSecretName, pubsubpluseventbroker)
				log.Info("Creating a new PreSharedAuthKey Secret", "Secret.Namespace", preSharedAuthKeySecret.Namespace, "Secret.Name", preSharedAuthKeySecret.Name)
				err = r.Create(ctx, preSharedAuthKeySecret)
				if err != nil {
					r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new PreSharedAuthKey Secret", "Secret.Namespace", preSharedAuthKeySecret.Namespace, "Secret.Name", preSharedAuthKeySecret.Name)
					return ctrl.Result{}, err
				}
				// PreSharedAuthKey Secret created successfully - return and requeue
				r.emitResourceSuccessEvent(pubsubpluseventbroker, "PreSharedAuthKey Secret", preSharedAuthSecretName)
				return ctrl.Result{Requeue: true}, nil
			} else if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get PreSharedAuthKey Secret")
				return ctrl.Result{}, err
			} else {
				log.V(1).Info("Detected existing PreSharedAuthKey Secret", " Secret.Name", preSharedAuthKeySecret.Name)
			}
		} else {
			err = r.Get(ctx, types.NamespacedName{Name: pubsubpluseventbroker.Spec.PreSharedAuthKeySecret, Namespace: pubsubpluseventbroker.Namespace}, preSharedAuthKeySecret)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to find specified PreSharedAuthKey Secret: '"+pubsubpluseventbroker.Spec.PreSharedAuthKeySecret+"'")
				return ctrl.Result{}, err
			} else {
				log.V(1).Info("Detected specified PreSharedAuthKey Secret", " Secret.Name", preSharedAuthKeySecret.Name)
			}
		}
	}

	// Check if Pod DisruptionBudget for HA  is Enabled, only when it is an HA deployment
	podDisruptionBudgetHAEnabled := pubsubpluseventbroker.Spec.PodDisruptionBudgetForHA
	if haDeployment && podDisruptionBudgetHAEnabled {
		// Check if PDB for HA already exists
		foundPodDisruptionBudgetHA := &policyv1.PodDisruptionBudget{}
		podDisruptionBudgetHAName := getObjectName("PodDisruptionBudget", pubsubpluseventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: podDisruptionBudgetHAName, Namespace: pubsubpluseventbroker.Namespace}, foundPodDisruptionBudgetHA)
		if err != nil && errors.IsNotFound(err) {
			//Pod DisruptionBudget for HA not available create new one
			podDisruptionBudgetHA := r.newPodDisruptionBudgetForHADeployment(podDisruptionBudgetHAName, pubsubpluseventbroker)
			log.Info("Creating new Pod Disruption Budget", "PodDisruptionBudget.Name", podDisruptionBudgetHAName)
			err = r.Create(ctx, podDisruptionBudgetHA)
			if err != nil {
				return ctrl.Result{}, err
			}
			// PodDisruptionBudget created successfully - return and requeue
			r.emitResourceSuccessEvent(pubsubpluseventbroker, "PodDisruptionBudget", podDisruptionBudgetHAName)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get PodDisruptionBudget")
			return ctrl.Result{}, err
		}
	}
	// TODO: add else branch to delete PDB if it existed to support dynamic update of PDB

	// At this point update service ready status and also report any pods that may be pending
	if _, err = r.getBrokerPod(ctx, pubsubpluseventbroker, Active); err != nil {
		// No active pod found, not service ready
		r.SetCondition(ctx, log, pubsubpluseventbroker, ServiceReadyCondition, metav1.ConditionFalse, WaitingForActivePodReason, "Waiting for active pod to provide broker service")
	} else {
		// Found active pod, service ready
		r.SetCondition(ctx, log, pubsubpluseventbroker, ServiceReadyCondition, metav1.ConditionTrue, ActivePodAndServiceExistsReason, "Found active broker pod and service exists")
	}
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(pubsubpluseventbroker.Namespace),
		client.MatchingLabels(baseLabels(pubsubpluseventbroker.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to list pods", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
		return ctrl.Result{}, err
	}
	for _, s := range podList.Items {
		if s.Status.Phase == corev1.PodPending {
			r.SetCondition(ctx, log, pubsubpluseventbroker, NoWarningsCondition, metav1.ConditionFalse, AtLeastOnePodPendingReason, "At least one pod is Pending, check for underlying issues if not transient")
			break
		}
	}

	// prep variables to be used next
	automatedPodUpdateStrategy := (pubsubpluseventbroker.Spec.UpdateStrategy != eventbrokerv1alpha1.ManualPodRestartUpdateStrategy)
	brokerSpecHash := brokerSpecHash(pubsubpluseventbroker.Spec)
	tlsSecretHash := r.tlsSecretHash(ctx, pubsubpluseventbroker)
	// Check if Primary StatefulSet already exists, if not create a new one
	stsP = &appsv1.StatefulSet{}
	stsPName := getStatefulsetName(pubsubpluseventbroker.Name, "p")
	err = r.Get(ctx, types.NamespacedName{Name: stsPName, Namespace: pubsubpluseventbroker.Namespace}, stsP)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		stsP := r.createStatefulsetForEventBroker(stsPName, ctx, pubsubpluseventbroker, sa, adminSecret, preSharedAuthKeySecret, monitoringSecret)
		log.Info("Creating a new Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
		err = r.Create(ctx, stsP)
		if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
			return ctrl.Result{}, err
		}
		// Primary StatefulSet created successfully - return and requeue
		r.emitResourceSuccessEvent(pubsubpluseventbroker, "Primary StatefulSet", stsPName)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Primary StatefulSet")
		return ctrl.Result{}, err
	}

	if haDeployment {
		// Add backup and monitor statefulsets
		// == Check if Backup StatefulSet already exists, if not create a new one
		stsB = &appsv1.StatefulSet{}
		stsBName := getStatefulsetName(pubsubpluseventbroker.Name, "b")
		err = r.Get(ctx, types.NamespacedName{Name: stsBName, Namespace: pubsubpluseventbroker.Namespace}, stsB)
		if err != nil && errors.IsNotFound(err) {
			// Define a new statefulset
			stsB := r.createStatefulsetForEventBroker(stsBName, ctx, pubsubpluseventbroker, sa, adminSecret, preSharedAuthKeySecret, monitoringSecret)
			log.Info("Creating a new Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
			err = r.Create(ctx, stsB)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
				return ctrl.Result{}, err
			}
			// Backup StatefulSet created successfully - return and requeue
			r.emitResourceSuccessEvent(pubsubpluseventbroker, "Backup StatefulSet", stsBName)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Backup StatefulSet")
			return ctrl.Result{}, err
		}

		// == Check if Monitor StatefulSet already exists, if not create a new one
		stsM = &appsv1.StatefulSet{}
		stsMName := getStatefulsetName(pubsubpluseventbroker.Name, "m")
		err = r.Get(ctx, types.NamespacedName{Name: stsMName, Namespace: pubsubpluseventbroker.Namespace}, stsM)
		if err != nil && errors.IsNotFound(err) {
			// Define a new statefulset
			stsM := r.createStatefulsetForEventBroker(stsMName, ctx, pubsubpluseventbroker, sa, adminSecret, preSharedAuthKeySecret, monitoringSecret)
			log.Info("Creating a new Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
			err = r.Create(ctx, stsM)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
				return ctrl.Result{}, err
			}
			// Monitor StatefulSet created successfully - return and requeue
			r.emitResourceSuccessEvent(pubsubpluseventbroker, "Monitor StatefulSet", stsMName)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Monitor StatefulSet")
			return ctrl.Result{}, err
		}
	}

	// Check and address if statefulsets require update
	if haDeployment {
		// Monitor
		if brokerStsOutdated(stsM, brokerSpecHash, tlsSecretHash) {
			log.Info("Updating existing Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
			r.updateStatefulsetForEventBroker(stsM, ctx, pubsubpluseventbroker, sa, adminSecret, preSharedAuthKeySecret, monitoringSecret)
			err = r.Update(ctx, stsM)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to update Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
				return ctrl.Result{}, err
			}
			// StatefulSet updated successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.V(1).Info("Detected up-to-date existing Monitor StatefulSet", " StatefulSet.Name", stsM.Name)
		// Backup
		if brokerStsOutdated(stsB, brokerSpecHash, tlsSecretHash) {
			log.Info("Updating existing Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
			r.updateStatefulsetForEventBroker(stsB, ctx, pubsubpluseventbroker, sa, adminSecret, preSharedAuthKeySecret, monitoringSecret)
			err = r.Update(ctx, stsB)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to update Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
				return ctrl.Result{}, err
			}
			// StatefulSet updated successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.V(1).Info("Detected up-to-date existing Backup StatefulSet", " StatefulSet.Name", stsB.Name)
	}
	// Primary (includes non-HA case)
	if brokerStsOutdated(stsP, brokerSpecHash, tlsSecretHash) {
		log.Info("Updating existing Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
		r.updateStatefulsetForEventBroker(stsP, ctx, pubsubpluseventbroker, sa, adminSecret, preSharedAuthKeySecret, monitoringSecret)
		err = r.Update(ctx, stsP)
		if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to update Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
			return ctrl.Result{}, err
		}
		// StatefulSet updated successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}
	log.V(1).Info("Detected up-to-date existing Primary StatefulSet", " StatefulSet.Name", stsP.Name)

	// At this point all required operator-managed broker artifacts are in place.

	// First check for readiness of all broker node pods to continue
	if stsP.Status.ReadyReplicas < 1 {
		log.Info("Detected unready Primary StatefulSet, waiting to be ready")
		if haDeployment {
			// Reset HAReady condition to false
			r.SetCondition(ctx, log, pubsubpluseventbroker, HAReadyCondition, metav1.ConditionFalse, MissingReadyPodReason, "Primary broker node is not HA ready")
		}
		return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
	}
	if haDeployment {
		if stsB.Status.ReadyReplicas < 1 {
			log.Info("Detected unready Backup StatefulSet, waiting to be ready")
			// Reset HAReady condition to false
			r.SetCondition(ctx, log, pubsubpluseventbroker, HAReadyCondition, metav1.ConditionFalse, MissingReadyPodReason, "Backup broker node is not HA ready")
			return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
		}
		if stsM.Status.ReadyReplicas < 1 {
			log.Info("Detected unready Monitor StatefulSet, waiting to be ready")
			// Reset HAReady condition to false
			r.SetCondition(ctx, log, pubsubpluseventbroker, HAReadyCondition, metav1.ConditionFalse, MissingReadyPodReason, "Monitor broker node is not HA ready")
			return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
		}
	}
	log.V(1).Info("All broker pods are in ready state")

	// At this point all statefulsets including their managed pods are ready

	// Next restart any out-of-sync broker pods to sync with their config dependencies
	// Skip it though if updateStrategy is set to manual - in this case this is supposed to be done manually by the user
	if automatedPodUpdateStrategy {
		var brokerPod *corev1.Pod
		// Must distinguish between HA and non-HA
		if haDeployment {
			// The algorithm is to process the Monitor, then the pod with `active=false`, finally `active=true`
			// == Monitor
			if brokerPod, err = r.getBrokerPod(ctx, pubsubpluseventbroker, Monitor); err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to list Monitor pod", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
				return ctrl.Result{}, err
			}
			if brokerPodOutdated(brokerPod, brokerSpecHash, tlsSecretHash) {
				if brokerPod.ObjectMeta.DeletionTimestamp == nil {
					// Restart the Monitor pod to sync with its Statefulset config
					log.Info("Monitor pod outdated, restarting to reflect latest updates", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
					err := r.Delete(ctx, brokerPod)
					if err != nil {
						r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to delete the Monitor pod", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
						return ctrl.Result{}, err
					}
					r.emitResourceRestartEvent(pubsubpluseventbroker, "Broker Pod", brokerPod.Name)
				}
				// Already restarting, just requeue
				return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
			}
			// == Standby
			if brokerPod, err = r.getBrokerPod(ctx, pubsubpluseventbroker, Standby); err != nil {
				log.Info("Failed to list a single Standby broker pod. Likely a temporary issue that more than one pod labelled active=false. Investigate if persists")
				// Just requeue
				return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
			}
			if brokerPodOutdated(brokerPod, brokerSpecHash, tlsSecretHash) {
				if brokerPod.ObjectMeta.DeletionTimestamp == nil {
					// Restart the Standby pod to sync with its Statefulset config
					log.Info("Standby pod outdated, restarting to reflect latest updates", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
					err := r.Delete(ctx, brokerPod)
					if err != nil {
						r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to delete the Standby pod", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
						return ctrl.Result{}, err
					}
					r.emitResourceRestartEvent(pubsubpluseventbroker, "Broker Pod", brokerPod.Name)
				}
				// Already restarting, just requeue
				return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
			}
		}
		// At this point, HA or not, check the active pod for restart
		if brokerPod, err = r.getBrokerPod(ctx, pubsubpluseventbroker, Active); err != nil {
			if haDeployment {
				// In case of HA it is expected that there is an active pod if control got this far
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to list the Active pod", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
				return ctrl.Result{}, err
			} else {
				// In case of non-HA this means that there is no active pod (likely restarting)
				// This is expected to be a temporary issue, just requeue
				return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
			}
		}
		if brokerPodOutdated(brokerPod, brokerSpecHash, tlsSecretHash) {
			if brokerPod.ObjectMeta.DeletionTimestamp == nil {
				// Restart the Active Pod to sync with its Statefulset config
				log.Info("Active pod outdated, restarting to reflect latest updates", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
				err := r.Delete(ctx, brokerPod)
				if err != nil {
					r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to delete the Active pod", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
					return ctrl.Result{}, err
				}
				r.emitResourceRestartEvent(pubsubpluseventbroker, "Broker Pod", brokerPod.Name)
			}
			// Already restarting, just requeue
			return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
		}
	}

	// At this point the broker is all up-to-date and healthy
	r.SetCondition(ctx, log, pubsubpluseventbroker, NoWarningsCondition, metav1.ConditionTrue, NoIssuesReason, "No issues to report")
	if haDeployment {
		r.SetCondition(ctx, log, pubsubpluseventbroker, HAReadyCondition, metav1.ConditionTrue, AllBrokersHAReadyInRedundancyGroupReason, "All broker nodes in the redundancy group are HA ready")
	}

	// Check and ensure setup if Prometheus Exporter is enabled
	prometheusExporterEnabled := pubsubpluseventbroker.Spec.Monitoring.Enabled
	prometheusExporterSvc := &corev1.Service{}
	prometheusExporterDeployment := &appsv1.Deployment{}
	if prometheusExporterEnabled {
		// Check if the Deployment to manage the Prometheus Exporter Pod already exists
		prometheusExporterDeploymentName := getObjectName("PrometheusExporterDeployment", pubsubpluseventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: prometheusExporterDeploymentName, Namespace: pubsubpluseventbroker.Namespace}, prometheusExporterDeployment)
		if err != nil && errors.IsNotFound(err) {
			//exporter not available create new one
			prometheusExporterDeployment = r.newDeploymentForPrometheusExporter(prometheusExporterDeploymentName, monitoringSecret, pubsubpluseventbroker)
			log.Info("Creating new Prometheus Exporter Deployment", "Deployment.Namespace", prometheusExporterDeployment.Namespace, "Deployment.Name", prometheusExporterDeploymentName)
			err = r.Create(ctx, prometheusExporterDeployment)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Prometheus Exporter Deployment", "Deployment.Namespace", prometheusExporterDeployment.Namespace, "Deployment.Name", prometheusExporterDeploymentName)
				return ctrl.Result{}, err
			}
			// Prometheus Exporter Deployment created successfully - return and requeue
			r.emitResourceSuccessEvent(pubsubpluseventbroker, "Prometheus Exporter Deployment", prometheusExporterDeploymentName)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Prometheus Exporter Deployment")
			return ctrl.Result{}, err
		}
		// Deployment already exists - don't requeue
		log.V(1).Info("Detected existing Prometheus Exporter Deployment", "Deployment.Name", prometheusExporterDeployment.Name)

		// Check if this Service for Prometheus Exporter Pod already exists
		prometheusExporterSvcName := getObjectName("PrometheusExporterService", pubsubpluseventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: prometheusExporterSvcName, Namespace: pubsubpluseventbroker.Namespace}, prometheusExporterSvc)
		if err != nil && errors.IsNotFound(err) {
			// New service for Prometheus Exporter
			prometheusExporterSvc = r.newServiceForPrometheusExporter(&pubsubpluseventbroker.Spec.Monitoring, prometheusExporterSvcName, pubsubpluseventbroker)
			log.Info("Creating a new Service for Prometheus Exporter", "Service.Namespace", prometheusExporterSvc.Namespace, "Service.Name", prometheusExporterSvc.Name)

			err = r.Create(ctx, prometheusExporterSvc)
			if err != nil {
				r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to create new Prometheus Exporter Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
				return ctrl.Result{}, err
			}
			// Prometheus Exporter Service created successfully - return and requeue
			r.emitResourceSuccessEvent(pubsubpluseventbroker, "Prometheus Exporter Service", prometheusExporterSvcName)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to get Prometheus Exporter Service")
			return ctrl.Result{}, err
		} else {
			log.V(1).Info("Detected existing Prometheus Exporter Service", " Service.Name", prometheusExporterSvc.Name)
		}
		// At this point monitoring is setup
		r.SetCondition(ctx, log, pubsubpluseventbroker, MonitoringReadyCondition, metav1.ConditionTrue, MonitoringReadyReason, "All checks passed")
	}

	// Now update elements of the PubSubPlusEventBroker deployment status; fetch the latest as needed
	err = r.Get(ctx, req.NamespacedName, pubsubpluseventbroker)
	if err != nil {
		// If any error then requeue, this will be handled at the beginning of the next reconcile
		return ctrl.Result{Requeue: true}, nil
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to list pods", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
		return ctrl.Result{}, err
	}
	pubsubpluseventbroker.Status.PodsList = getPodNames(podList.Items)
	statefulSets := []string{stsP.Name}
	if haDeployment {
		statefulSets = append(statefulSets, stsB.Name, stsM.Name)
	}
	var brokerImage string
	for _, s := range stsP.Spec.Template.Spec.Containers {
		if s.Name == "pubsubplus" {
			brokerImage = s.Image
		}
	}
	pubsubpluseventbroker.Status.Broker = eventbrokerv1alpha1.BrokerSubStatus{
		HADeployment:           strconv.FormatBool(haDeployment),
		TLSSupport:             strconv.FormatBool(pubsubpluseventbroker.Spec.BrokerTLS.Enabled),
		TLSSecret:              pubsubpluseventbroker.Spec.BrokerTLS.ServerTLsConfigSecret,
		AdminCredentialsSecret: adminSecret.Name,
		ServiceName:            svc.Name,
		ServiceType:            string(svc.Spec.Type),
		StatefulSets:           statefulSets,
		BrokerImage:            brokerImage,
	}
	pubsubpluseventbroker.Status.Monitoring = eventbrokerv1alpha1.MonitoringSubStatus{
		Enabled: strconv.FormatBool(prometheusExporterEnabled),
	}
	if prometheusExporterEnabled {
		pubsubpluseventbroker.Status.Monitoring.ServiceName = prometheusExporterSvc.Name
		pubsubpluseventbroker.Status.Monitoring.ExporterImage = prometheusExporterDeployment.Spec.Template.Spec.Containers[0].Image
	}
	err = r.Status().Update(ctx, pubsubpluseventbroker)
	if err != nil {
		if errors.IsConflict(err) {
			// Requeue to try again with with refreshed resource
			return ctrl.Result{Requeue: true}, nil
		}
		// log any other error
		r.recordErrorState(ctx, log, pubsubpluseventbroker, err, ResourceErrorReason, "Failed to update PubSubPlusEventBroker status")
		return ctrl.Result{}, err
	}

	// When finished still reconcile periodically to wake up periodically
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil

} // End of Reconcile

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// emitResourceSuccessEvent
func (r *PubSubPlusEventBrokerReconciler) emitResourceSuccessEvent(pubsubpluseventbroker *eventbrokerv1alpha1.PubSubPlusEventBroker, resourceType string, resourceName string) {
	r.Recorder.Event(pubsubpluseventbroker, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", resourceType, resourceName))
}

// emitResourceRestartEvent
func (r *PubSubPlusEventBrokerReconciler) emitResourceRestartEvent(pubsubpluseventbroker *eventbrokerv1alpha1.PubSubPlusEventBroker, resourceType string, resourceName string) {
	r.Recorder.Event(pubsubpluseventbroker, corev1.EventTypeNormal, "Restarted", fmt.Sprintf("Restarted outdated %s %s", resourceType, resourceName))
}

// recordErrorState is the central point to log, emit event and set warning status condition if an error has been detected
func (r *PubSubPlusEventBrokerReconciler) recordErrorState(ctx context.Context, log logr.Logger, pubsubpluseventbroker *eventbrokerv1alpha1.PubSubPlusEventBroker, err error, reason ConditionReason, msg string, keysAndValues ...interface{}) {
	log.Error(err, msg, keysAndValues...)
	r.Recorder.Event(pubsubpluseventbroker, corev1.EventTypeWarning, string(reason), msg)
	r.SetCondition(ctx, log, pubsubpluseventbroker, NoWarningsCondition, metav1.ConditionFalse, reason, msg)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PubSubPlusEventBrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Need to watch non-managed resources
	// To understand following code refer to https://kubebuilder.io/reference/watching-resources/externally-managed.html#allow-for-linking-of-resources-in-the-spec
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &eventbrokerv1alpha1.PubSubPlusEventBroker{}, dependencyTlsSecretField, func(rawObj client.Object) []string {
		// Extract the secret name from the EventBroker Spec, if one is provided
		eventBroker := rawObj.(*eventbrokerv1alpha1.PubSubPlusEventBroker)
		if eventBroker.Spec.BrokerTLS.ServerTLsConfigSecret == "" {
			return nil
		}
		return []string{eventBroker.Spec.BrokerTLS.ServerTLsConfigSecret}
	}); err != nil {
		return err
	}
	// This describes rules to manage both owned and external reources
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventbrokerv1alpha1.PubSubPlusEventBroker{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.reconcileRequestsForEventBrokersDependingOnTlsSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *PubSubPlusEventBrokerReconciler) reconcileRequestsForEventBrokersDependingOnTlsSecret(secret client.Object) []reconcile.Request {
	ebDeployments := &eventbrokerv1alpha1.PubSubPlusEventBrokerList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(dependencyTlsSecretField, secret.GetName()),
		Namespace:     secret.GetNamespace(),
	}
	err := r.List(context.TODO(), ebDeployments, listOps)
	if err != nil {
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, len(ebDeployments.Items))
	for i, item := range ebDeployments.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}
