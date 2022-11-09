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
)

// PubSubPlusEventBrokerReconciler reconciles a PubSubPlusEventBroker object
type PubSubPlusEventBrokerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

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
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get PubSubPlusEventBroker")
		return ctrl.Result{}, err
	} else {
		log.Info("Detected existing pubsubpluseventbroker", " pubsubpluseventbroker.Name", pubsubpluseventbroker.Name)
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
	svc := &corev1.Service{}
	svcName := getObjectName("Service", pubsubpluseventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: pubsubpluseventbroker.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		svc := r.serviceForEventBroker(svcName, pubsubpluseventbroker)
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
		log.Info("Detected existing Service", " Service.Name", svc.Name)
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

	// Check if Primary StatefulSet already exists, if not create a new one
	stsP = &appsv1.StatefulSet{}
	stsPName := getStatefulsetName(pubsubpluseventbroker.Name, "p")
	err = r.Get(ctx, types.NamespacedName{Name: stsPName, Namespace: pubsubpluseventbroker.Namespace}, stsP)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		stsP := r.createStatefulsetForEventBroker(stsPName, pubsubpluseventbroker, sa)
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
		if stsP.Spec.Template.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != hash(pubsubpluseventbroker.Spec) {
			// If resource versions differ it means update is required
			log.Info("Updating existing Primary StatefulSet", " StatefulSet.Name", stsP.Name)
			r.updateStatefulsetForEventBroker(stsPName, pubsubpluseventbroker, stsP, sa)
			log.Info("Updating Primary StatefulSet", "StatefulSet.Namespace", stsP.Namespace, "StatefulSet.Name", stsP.Name)
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
			stsB := r.createStatefulsetForEventBroker(stsBName, pubsubpluseventbroker, sa)
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
			if stsB.Spec.Template.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != hash(pubsubpluseventbroker.Spec) {
				// If resource versions differ it means update is required
				log.Info("Updating existing Backup StatefulSet", " StatefulSet.Name", stsB.Name)
				r.updateStatefulsetForEventBroker(stsBName, pubsubpluseventbroker, stsB, sa)
				log.Info("Updating Backup StatefulSet", "StatefulSet.Namespace", stsB.Namespace, "StatefulSet.Name", stsB.Name)
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
			stsM := r.createStatefulsetForEventBroker(stsMName, pubsubpluseventbroker, sa)
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
			if stsM.Spec.Template.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != hash(pubsubpluseventbroker.Spec) {
				// If resource versions differ it means update is required
				log.Info("Updating existing Monitor StatefulSet", " StatefulSet.Name", stsM.Name)
				r.updateStatefulsetForEventBroker(stsMName, pubsubpluseventbroker, stsM, sa)
				log.Info("Updating Monitor StatefulSet", "StatefulSet.Namespace", stsM.Namespace, "StatefulSet.Name", stsM.Name)
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

	// Check if pods are out-of-sync and need to be restarted
	// First check for readiness of all broker nodes to continue
	// TODO: where it makes sense emit events here instead of logs
	if stsP.Status.ReadyReplicas < 1 {
		log.Info("Detected unready Primary StatefulSet, waiting to be ready")
		return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
	} else if pubsubpluseventbroker.Spec.Redundancy {
		if stsB.Status.ReadyReplicas < 1 {
			log.Info("Detected unready Backup StatefulSet, waiting to be ready")
			return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
		}
		if stsM.Status.ReadyReplicas < 1 {
			log.Info("Detected unready Monitor StatefulSet, waiting to be ready")
			return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
		}
	}
	log.Info("All broker pods are available")

	// Next restart any pods to sync with their config dependencies
	expectedConfigSignature := hash(pubsubpluseventbroker.Spec)
	var brokerPod *corev1.Pod
	// Must distinguish between HA and non-HA
	if haDeployment {
		// The algorithm is to process the Monitor, then the pod with `active=false`, finally `active=true`
		// == Monitor
		if brokerPod, err = r.getBrokerPod(ctx, pubsubpluseventbroker, Monitor); err != nil {
			log.Error(err, "Failed to list pods", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
			return ctrl.Result{}, err
		}
		if brokerPod.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != expectedConfigSignature {
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
			log.Error(err, "Failed to list pods", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
			return ctrl.Result{}, err
		}
		if brokerPod.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != expectedConfigSignature {
			if brokerPod.ObjectMeta.DeletionTimestamp == nil {
				// Restart the Standby pod to sync with its Statefulset config
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
		log.Error(err, "Failed to list pods", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
		return ctrl.Result{}, err
	}
	if brokerPod.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != expectedConfigSignature {
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

	// List the pods for this pubsubpluseventbroker
	// podList := &corev1.PodList{}
	// listOpts := []client.ListOption{
	// 	client.InNamespace(pubsubpluseventbroker.Namespace),
	// 	client.MatchingLabels(getMessagingPodSelectorByActive(pubsubpluseventbroker.Name, "true")),
	// }
	// if err = r.List(ctx, podList, listOpts...); err != nil {
	// 	log.Error(err, "Failed to list pods", "PubSubPlusEventBroker.Namespace", pubsubpluseventbroker.Namespace, "PubSubPlusEventBroker.Name", pubsubpluseventbroker.Name)
	// 	return ctrl.Result{}, err
	// }
	// if (podList != nil && len(podList.Items) > 0 &&
	// 	podList.Items[0].ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != stsP.Spec.Template.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] &&
	// 	podList.Items[0].ObjectMeta.DeletionTimestamp == nil) {
	// 	// Restart the Pod to sync with its Statefulset config
	// 	log.Info("Restarting pod the reflect latest updates", "Pod.Namespace", &podList.Items[0].Namespace, "Pod.Name", &podList.Items[0].Name)
	// 	err = r.Delete(ctx, &podList.Items[0])
	// 	if err != nil {
	// 		log.Error(err, "Failed to delete the Pod", "Pod.Namespace", &podList.Items[0].Namespace, "Pod.Name", &podList.Items[0].Name)
	// 		return ctrl.Result{}, err
	// 	}
	// 	// Now wait for the pod to come back up
	// 	return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
	// }

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
		err := r.Status().Update(ctx, pubsubpluseventbroker)
		if err != nil {
			log.Error(err, "Failed to update PubSubPlusEventBroker status")
			return ctrl.Result{}, err
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
		log.Info("Skip reconcile: Deployment already exists", "Pod.Namespace", foundPrometheusExporter.Namespace, "Pod.Name", foundPrometheusExporter.Name)

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

	return ctrl.Result{}, nil
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventbrokerv1alpha1.PubSubPlusEventBroker{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
