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
	"reflect"
	"strings"
	"time"

	policyv1 "k8s.io/api/policy/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"

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
	Scheme *runtime.Scheme
}

const (
	dependencyTlsSecretField = ".spec.tls.serverTlsConfigSecret"
)

// TODO: review and revise to minimum at the end of the dev cycle!
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PubSubPlusEventBroker object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *PubSubPlusEventBrokerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Safeguards in case reconcile gets stuck - not used for now
	// ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	// defer cancel()
	// Format is set in main.go
	// TODO: better share logger within module so code in other source files can make use of logging too
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
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get PubSubPlusEventBroker")
		return ctrl.Result{}, err
	} else {
		log.Info("Detected existing pubsubpluseventbroker", " pubsubpluseventbroker.Name", pubsubpluseventbroker.Name)
	}

	// Check maintenance mode
	if labelValue, ok := pubsubpluseventbroker.Labels[maintenanceLabel]; ok && labelValue == "true" {
		log.Info(fmt.Sprintf("Found maintenance label '%s=true', reconcile paused.", maintenanceLabel))
		// TODO: update status
		return ctrl.Result{}, nil
	}

	// Check if new ServiceAccount needs to created or an existing one needs to be used
	sa := &corev1.ServiceAccount{}
	if len(strings.TrimSpace(pubsubpluseventbroker.Spec.ServiceAccount.Name)) > 0 {
		err = r.Get(ctx, types.NamespacedName{Name: pubsubpluseventbroker.Spec.ServiceAccount.Name, Namespace: pubsubpluseventbroker.Namespace}, sa)
		if err != nil && errors.IsNotFound(err) {
			log.Error(err, "Failed to find existing ServiceAccount", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", pubsubpluseventbroker.Spec.ServiceAccount.Name)
			return ctrl.Result{}, err
		}
		log.Info("Found existing ServiceAccount", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", sa.Name)
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
				log.Error(err, "Failed to create new ServiceAccount", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", sa.Name)
				return ctrl.Result{}, err
			}
			// ServiceAccount created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			log.Error(err, "Failed to get ServiceAccount")
			return ctrl.Result{}, err
		} else {
			// TODO: this should be Debug level... but it seems there is no log.Debug ! Need to investigate how to do Debug log
			log.Info("Detected existing ServiceAccount", " ServiceAccount.Name", sa.Name)
		}
	}

	// Check if podtagupdater Role already exists, if not create a new one
	role := &rbacv1.Role{}
	roleName := getObjectName("Role", pubsubpluseventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: roleName, Namespace: pubsubpluseventbroker.Namespace}, role)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Role
		role := r.roleForEventBroker(roleName, pubsubpluseventbroker)
		log.Info("Creating a new Role", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
		err = r.Create(ctx, role)
		if err != nil {
			log.Error(err, "Failed to create new Role", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
			return ctrl.Result{}, err
		}
		// Role created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Role")
		return ctrl.Result{}, err
	} else {
		log.Info("Detected existing Role", " Role.Name", role.Name)
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
			log.Error(err, "Failed to create new RoleBinding", "RoleBinding.Namespace", rb.Namespace, "RoleBinding.Name", rb.Name)
			return ctrl.Result{}, err
		}
		// RoleBinding created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get RoleBinding")
		return ctrl.Result{}, err
	} else {
		log.Info("Detected existing RoleBinding", " RoleBinding.Name", rb.Name)
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
			log.Error(err, "Failed to create new ConfigMap", "Configmap.Namespace", cm.Namespace, "Configmap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		// ConfigMap created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	} else {
		log.Info("Detected existing ConfigMap", " ConfigMap.Name", cm.Name)
	}

	// Check if the Service already exists, if not create a new one
	brokerServiceHash := brokerServiceHash(pubsubpluseventbroker.Spec)
	svc := &corev1.Service{}
	svcName := getObjectName("Service", pubsubpluseventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: pubsubpluseventbroker.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		svc := r.createServiceForEventBroker(svcName, pubsubpluseventbroker)
		log.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	} else {
		// Check if existing service is not outdated at this point
		if brokerServiceOutdated(svc, brokerServiceHash) {
			log.Info("Updating existing Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			r.updateServiceForEventBroker(svc, pubsubpluseventbroker)
			err = r.Update(ctx, svc)
			if err != nil {
				log.Error(err, "Failed to update Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
				return ctrl.Result{}, err
			}
			// Service updated successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Info("Detected up-to-date existing Service", " Service.Name", svc.Name)
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
			log.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			err = r.Create(ctx, svc)
			if err != nil {
				log.Error(err, "Failed to create new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
				return ctrl.Result{}, err
			}
			// Service created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			log.Error(err, "Failed to get Service")
			return ctrl.Result{}, err
		} else {
			log.Info("Detected existing Discovery Service", " Service.Name", dsvc.Name)
		}
	}

	// Check Secret
	secret := &corev1.Secret{}
	secretName := getObjectName("Secret", pubsubpluseventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: pubsubpluseventbroker.Namespace}, secret)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Secret
		secret := r.secretForEventBroker(secretName, pubsubpluseventbroker)
		log.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Create(ctx, secret)
		if err != nil {
			log.Error(err, "Failed to create new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
			return ctrl.Result{}, err
		}
		// Secret created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Secret")
		return ctrl.Result{}, err
	} else {
		log.Info("Detected existing Secret", " Secret.Name", secret.Name)
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
			// Pod Disruption Budget created successfully - return requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			return ctrl.Result{}, err
		}
	}

	brokerSpecHash := brokerSpecHash(pubsubpluseventbroker.Spec)
	tlsSecretHash := r.tlsSecretHash(ctx, pubsubpluseventbroker)
	automatedPodUpdateStrategy := (pubsubpluseventbroker.Spec.UpdateStrategy != eventbrokerv1alpha1.ManualPodRestartUpdateStrategy)
	// Check if Primary StatefulSet already exists, if not create a new one
	stsP = &appsv1.StatefulSet{}
	stsPName := getStatefulsetName(pubsubpluseventbroker.Name, "p")
	err = r.Get(ctx, types.NamespacedName{Name: stsPName, Namespace: pubsubpluseventbroker.Namespace}, stsP)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		stsP := r.createStatefulsetForEventBroker(stsPName, ctx, pubsubpluseventbroker, sa)
		log.Info("Creating a new Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
		err = r.Create(ctx, stsP)
		if err != nil {
			log.Error(err, "Failed to create new Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
			return ctrl.Result{}, err
		}
		// StatefulSet created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get StatefulSet")
		return ctrl.Result{}, err
	} else {
		if brokerStsOutdated(stsP, brokerSpecHash, tlsSecretHash) {
			// If resource versions differ it means recreate or update is required
			if stsP.Status.ReadyReplicas < 1 {
				// Related pod is still starting up but already outdated. Remove this statefulset (if automated pod update is allowed and delete has not already been initiated)
				// and let's recreate it from scratch to force pod restart
				if automatedPodUpdateStrategy && stsP.ObjectMeta.DeletionTimestamp == nil {
					log.Info("Existing Primary StatefulSet requires update and its Pod is outdated and not ready - recreating StatefulSet to force pod update", " StatefulSet.Name", stsP.Name)
					err = r.Delete(ctx, stsP)
					if err != nil {
						log.Error(err, "Failed to delete Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
						return ctrl.Result{}, err
					}
					// StatefulSet deleted successfully - return and requeue
					return ctrl.Result{RequeueAfter: time.Duration(1) * time.Second}, nil
				}
				// Otherwise just continue
			}
			log.Info("Updating existing Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
			r.updateStatefulsetForEventBroker(stsP, ctx, pubsubpluseventbroker, sa)
			err = r.Update(ctx, stsP)
			if err != nil {
				log.Error(err, "Failed to update Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
				return ctrl.Result{}, err
			}
			// StatefulSet updated successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Info("Detected up-to-date existing Primary StatefulSet", " StatefulSet.Name", stsP.Name)
	}

	if haDeployment {
		// Add backup and monitor statefulsets
		// == Check if Backup StatefulSet already exists, if not create a new one
		stsB = &appsv1.StatefulSet{}
		stsBName := getStatefulsetName(pubsubpluseventbroker.Name, "b")
		err = r.Get(ctx, types.NamespacedName{Name: stsBName, Namespace: pubsubpluseventbroker.Namespace}, stsB)
		if err != nil && errors.IsNotFound(err) {
			// Define a new statefulset
			stsB := r.createStatefulsetForEventBroker(stsBName, ctx, pubsubpluseventbroker, sa)
			log.Info("Creating a new Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
			err = r.Create(ctx, stsB)
			if err != nil {
				log.Error(err, "Failed to create new Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
				return ctrl.Result{}, err
			}
			// StatefulSet created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			log.Error(err, "Failed to get StatefulSet")
			return ctrl.Result{}, err
		} else {
			if brokerStsOutdated(stsB, brokerSpecHash, tlsSecretHash) {
				// If resource versions differ it means recreate or update is required
				if stsB.Status.ReadyReplicas < 1 {
					// Related pod is still starting up but already outdated. Remove this statefulset (if automated pod update is allowed and delete has not already been initiated)
					// and let's recreate it from scratch to force pod restart
					if automatedPodUpdateStrategy && stsB.ObjectMeta.DeletionTimestamp == nil {
						log.Info("Existing Backup StatefulSet requires update and its Pod is outdated and not ready - recreating StatefulSet to force pod update", " StatefulSet.Name", stsB.Name)
						err = r.Delete(ctx, stsB)
						if err != nil {
							log.Error(err, "Failed to delete Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
							return ctrl.Result{}, err
						}
						// StatefulSet deleted successfully - return and requeue
						return ctrl.Result{RequeueAfter: time.Duration(1) * time.Second}, nil
					}
					// Otherwise just continue
				}
				log.Info("Updating existing Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
				r.updateStatefulsetForEventBroker(stsB, ctx, pubsubpluseventbroker, sa)
				err = r.Update(ctx, stsB)
				if err != nil {
					log.Error(err, "Failed to update Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
					return ctrl.Result{}, err
				}
				// StatefulSet updated successfully - return and requeue
				return ctrl.Result{Requeue: true}, nil
			}
			log.Info("Detected up-to-date existing Backup StatefulSet", " StatefulSet.Name", stsB.Name)
		}

		// == Check if Monitor StatefulSet already exists, if not create a new one
		stsM = &appsv1.StatefulSet{}
		stsMName := getStatefulsetName(pubsubpluseventbroker.Name, "m")
		err = r.Get(ctx, types.NamespacedName{Name: stsMName, Namespace: pubsubpluseventbroker.Namespace}, stsM)
		if err != nil && errors.IsNotFound(err) {
			// Define a new statefulset
			stsM := r.createStatefulsetForEventBroker(stsMName, ctx, pubsubpluseventbroker, sa)
			log.Info("Creating a new Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
			err = r.Create(ctx, stsM)
			if err != nil {
				log.Error(err, "Failed to create new Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
				return ctrl.Result{}, err
			}
			// StatefulSet created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			log.Error(err, "Failed to get StatefulSet")
			return ctrl.Result{}, err
		} else {
			if brokerStsOutdated(stsM, brokerSpecHash, tlsSecretHash) {
				// If resource versions differ it means recreate or update is required
				if stsM.Status.ReadyReplicas < 1 {
					// Related pod is still starting up but already outdated. Remove this statefulset (if automated pod update is allowed and delete has not already been initiated)
					// and let's recreate it from scratch to force pod restart
					if automatedPodUpdateStrategy && stsM.ObjectMeta.DeletionTimestamp == nil {
						log.Info("Existing Monitor StatefulSet requires update and its Pod is outdated and not ready - recreating StatefulSet to force pod update", " StatefulSet.Name", stsM.Name)
						err = r.Delete(ctx, stsM)
						if err != nil {
							log.Error(err, "Failed to delete Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
							return ctrl.Result{}, err
						}
						// StatefulSet deleted successfully - return and requeue
						return ctrl.Result{RequeueAfter: time.Duration(1) * time.Second}, nil
					}
					// Otherwise just continue
				}
				log.Info("Updating existing Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
				r.updateStatefulsetForEventBroker(stsM, ctx, pubsubpluseventbroker, sa)
				err = r.Update(ctx, stsM)
				if err != nil {
					log.Error(err, "Failed to update Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
					return ctrl.Result{}, err
				}
				// StatefulSet updated successfully - return and requeue
				return ctrl.Result{Requeue: true}, nil
			}
			log.Info("Detected up-to-date existing Monitor StatefulSet", " StatefulSet.Name", stsM.Name)
		}

	}

	// Check if pods are out-of-sync and need to be restarted
	// First check for readiness of all broker nodes to continue
	// TODO: where it makes sense emit events here instead of logs
	if stsP.Status.ReadyReplicas < 1 {
		log.Info("Detected unready Primary StatefulSet, waiting to be ready")
		return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
	}
	if haDeployment {
		if stsB.Status.ReadyReplicas < 1 {
			log.Info("Detected unready Backup StatefulSet, waiting to be ready")
			return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
		}
		if stsM.Status.ReadyReplicas < 1 {
			log.Info("Detected unready Monitor StatefulSet, waiting to be ready")
			return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
		}
	}
	log.Info("All broker pods are in ready state")

	// Next restart any pods to sync with their config dependencies
	// Skip it though if updateStrategy is set to manual - in this case this is supposed to be done manually by the user
	if automatedPodUpdateStrategy {
		var brokerPod *corev1.Pod
		// Must distinguish between HA and non-HA
		if haDeployment {
			// The algorithm is to process the Monitor, then the pod with `active=false`, finally `active=true`
			// == Monitor
			if brokerPod, err = r.getBrokerPod(ctx, pubsubpluseventbroker, Monitor); err != nil {
				log.Error(err, "Failed to list Monitor pod", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
				return ctrl.Result{}, err
			}
			if brokerPodOutdated(brokerPod, brokerSpecHash, tlsSecretHash) {
				if brokerPod.ObjectMeta.DeletionTimestamp == nil {
					// Restart the Monitor pod to sync with its Statefulset config
					log.Info("Restarting Monitor pod to reflect latest updates", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
					err := r.Delete(ctx, brokerPod)
					if err != nil {
						log.Error(err, "Failed to delete the Monitor pod", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
						return ctrl.Result{}, err
					}
				}
				// Already restarting, just requeue
				return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
			}
			// == Standby
			if brokerPod, err = r.getBrokerPod(ctx, pubsubpluseventbroker, Standby); err != nil {
				log.Error(err, "Failed to list a single Standby pod", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
				// This may be a temporary issue, most likely more than one pod labelled	 active=false, just requeue
				return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
			}
			if brokerPodOutdated(brokerPod, brokerSpecHash, tlsSecretHash) {
				if brokerPod.ObjectMeta.DeletionTimestamp == nil {
					// Restart the Standby pod to sync with its Statefulset config
					// TODO: it may be a better idea to let control come here even if
					//   automatedPodUpdateStrategy is not set. Then log the need for Pod restart.
					log.Info("Restarting Standby pod to reflect latest updates", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
					err := r.Delete(ctx, brokerPod)
					if err != nil {
						log.Error(err, "Failed to delete the Standby pod", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
						return ctrl.Result{}, err
					}
				}
				// Already restarting, just requeue
				return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
			}
		}
		// At this point, HA or not, check the active pod for restart
		if brokerPod, err = r.getBrokerPod(ctx, pubsubpluseventbroker, Active); err != nil {
			if haDeployment {
				// In case of HA it is expected that there is an active pod if control got this far
				log.Error(err, "Failed to list the Active pod", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
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
				log.Info("Restarting Active pod to reflect latest updates", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
				err := r.Delete(ctx, brokerPod)
				if err != nil {
					log.Error(err, "Failed to delete the Active pod", "Pod.Namespace", &brokerPod.Namespace, "Pod.Name", &brokerPod.Name)
					return ctrl.Result{}, err
				}
			}
			// Already restarting, just requeue
			return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
		}
	}

	// Check if Prometheus Exporter is enabled only after broker is running perfectly
	prometheusExporterEnabled := pubsubpluseventbroker.Spec.Monitoring.Enabled
	if prometheusExporterEnabled {
		// Check if this Prometheus Exporter Pod already exists
		foundPrometheusExporter := &appsv1.Deployment{}
		prometheusExporterName := getObjectName("PrometheusExporterDeployment", pubsubpluseventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: prometheusExporterName, Namespace: pubsubpluseventbroker.Namespace}, foundPrometheusExporter)
		if err != nil && errors.IsNotFound(err) {

			//exporter not available create new one
			prometheusExporter := r.newDeploymentForPrometheusExporter(prometheusExporterName, secret, pubsubpluseventbroker)

			log.Info("Creating new Prometheus Exporter", "Pod.Namespace", prometheusExporter.Namespace, "Pod.Name", prometheusExporterName)
			err = r.Create(ctx, prometheusExporter)

			if err != nil {
				return ctrl.Result{}, err
			}
			// Deployment created successfully - return requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			return ctrl.Result{}, err
		}

		// Pod already exists - don't requeue
		log.Info("Detected existing Prometheus Exporter deployment", "Deployment.Namespace", foundPrometheusExporter.Namespace, "Deployment.Name", foundPrometheusExporter.Name)

		// Check if this Service for Prometheus Exporter Pod already exists
		foundPrometheusExporterSvc := &corev1.Service{}
		prometheusExporterSvcName := getObjectName("PrometheusExporterService", pubsubpluseventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: prometheusExporterSvcName, Namespace: pubsubpluseventbroker.Namespace}, foundPrometheusExporterSvc)
		if err != nil && errors.IsNotFound(err) {
			// New service for Prometheus Exporter
			prometheusExporterSvc := r.newServiceForPrometheusExporter(pubsubpluseventbroker.Spec.Monitoring, prometheusExporterSvcName, pubsubpluseventbroker)
			log.Info("Creating a new Service for Prometheus Exporter", "Service.Namespace", prometheusExporterSvc.Namespace, "Service.Name", prometheusExporterSvc.Name)

			err = r.Create(ctx, prometheusExporterSvc)
			if err != nil {
				log.Error(err, "Failed to create new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
				return ctrl.Result{}, err
			}
			// Service created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			log.Error(err, "Failed to get Service")
			return ctrl.Result{}, err
		} else {
			log.Info("Detected existing Service", " Service.Name", svc.Name)
		}
		return ctrl.Result{}, nil
	}

	// Update the PubSubPlusEventBroker status with the pod names
	// TODO: this is an example. It would make sense to update status with broker ready for messaging, config update on progress, etc.
	// List the pods for this pubsubpluseventbroker's StatefulSet
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(pubsubpluseventbroker.Namespace),
		client.MatchingLabels(baseLabels(pubsubpluseventbroker.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)
	// Update status.BrokerPods if needed
	if !reflect.DeepEqual(podNames, pubsubpluseventbroker.Status.BrokerPods) {
		pubsubpluseventbroker.Status.BrokerPods = podNames
		// Get the resource first to ensure updating the latest
		err := r.Get(ctx, req.NamespacedName, pubsubpluseventbroker)
		if err == nil {
			err := r.Status().Update(ctx, pubsubpluseventbroker)
			if err != nil {
				log.Error(err, "Failed to update PubSubPlusEventBroker status")
				return ctrl.Result{}, err
			}
		}
		// if err wasn't nil then let the next reconcile loop handle it
	}

	// Reconcile periodically
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

// TODO: if still needed move it to namings
// baseLabels returns the labels for selecting the resources
// belonging to the given pubsubpluseventbroker CR name.
func baseLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance": name,
		"app.kubernetes.io/name":     appKubernetesIoNameLabel,
	}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
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
			handler.EnqueueRequestsFromMapFunc(r.findEventBrokersForTlsSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *PubSubPlusEventBrokerReconciler) findEventBrokersForTlsSecret(secret client.Object) []reconcile.Request {
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
