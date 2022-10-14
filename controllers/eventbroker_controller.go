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
	"time"

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

// EventBrokerReconciler reconciles a EventBroker object
type EventBrokerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// TODO: review and revise to minimum at the end of the dev cycle!
//+kubebuilder:rbac:groups=pubsubplus.solace.com,resources=eventbrokers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pubsubplus.solace.com,resources=eventbrokers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pubsubplus.solace.com,resources=eventbrokers/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
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
// the EventBroker object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *EventBrokerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// TODO: fix timestamp format
	log := ctrllog.FromContext(ctx)

	var stsP, stsB, stsM *appsv1.StatefulSet

	// Fetch the EventBroker instance
	eventbroker := &eventbrokerv1alpha1.EventBroker{}
	err := r.Get(ctx, req.NamespacedName, eventbroker)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("EventBroker resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get EventBroker")
		return ctrl.Result{}, err
	} else {
		log.Info("Detected existing eventbroker", " eventbroker.Name", eventbroker.Name)
	}

	// Check if ServiceAccount already exists, if not create a new one
	sa := &corev1.ServiceAccount{}
	saName := getObjectName("ServiceAccount", eventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: saName, Namespace: eventbroker.Namespace}, sa)
	if err != nil && errors.IsNotFound(err) {
		// Define a new ServiceAccount
		sa := r.serviceaccountForEventBroker(saName, eventbroker)
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
		log.Info("Detected existing ServiceAccount", " ServiceAccount.Name", sa.Name)
	}

	// Check if podtagupdater Role already exists, if not create a new one
	role := &rbacv1.Role{}
	roleName := getObjectName("Role", eventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: roleName, Namespace: eventbroker.Namespace}, role)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Role
		role := r.roleForEventBroker(roleName, eventbroker)
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
	rbName := getObjectName("RoleBinding", eventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: rbName, Namespace: eventbroker.Namespace}, rb)
	if err != nil && errors.IsNotFound(err) {
		// Define a new RoleBinding
		rb := r.rolebindingForEventBroker(rbName, eventbroker)
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
	cmName := getObjectName("ConfigMap", eventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: eventbroker.Namespace}, cm)
	if err != nil && errors.IsNotFound(err) {
		// Define a new configmap
		cm := r.configmapForEventBroker(cmName, eventbroker)
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
	svcName := getObjectName("Service", eventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: eventbroker.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		svc := r.serviceForEventBroker(svcName, eventbroker)
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

	haDeployment := eventbroker.Spec.Redundancy
	if haDeployment {
		// Check if the Discovery Service already exists, if not create a new one
		dsvc := &corev1.Service{}
		dsvcName := getObjectName("DiscoveryService", eventbroker.Name)
		err = r.Get(ctx, types.NamespacedName{Name: dsvcName, Namespace: eventbroker.Namespace}, dsvc)
		if err != nil && errors.IsNotFound(err) {
			// Define a new service
			svc := r.discoveryserviceForEventBroker(dsvcName, eventbroker)
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
	secretName := getObjectName("Secret", eventbroker.Name)
	err = r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: eventbroker.Namespace}, secret)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Secret
		secret := r.secretForEventBroker(secretName, eventbroker)
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
	stsPName := getStatefulsetName(eventbroker.Name, "p")
	err = r.Get(ctx, types.NamespacedName{Name: stsPName, Namespace: eventbroker.Namespace}, stsP)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		stsP := r.createStatefulsetForEventBroker(stsPName, eventbroker)
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
		if stsP.Spec.Template.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != hash(eventbroker.Spec) {
			// If resource versions differ it means update is required
			log.Info("Updating existing Primary StatefulSet", " StatefulSet.Name", stsP.Name)
			r.updateStatefulsetForEventBroker(stsPName, eventbroker, stsP)
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

		// Check if Backup StatefulSet already exists, if not create a new one
		stsB = &appsv1.StatefulSet{}
		stsBName := getStatefulsetName(eventbroker.Name, "b")
		err = r.Get(ctx, types.NamespacedName{Name: stsBName, Namespace: eventbroker.Namespace}, stsB)
		if err != nil && errors.IsNotFound(err) {
			// Define a new statefulset
			stsB := r.createStatefulsetForEventBroker(stsBName, eventbroker)
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
			if stsB.Spec.Template.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != hash(eventbroker.Spec) {
				// If resource versions differ it means update is required
				log.Info("Updating existing Backup StatefulSet", " StatefulSet.Name", stsB.Name)
				r.updateStatefulsetForEventBroker(stsBName, eventbroker, stsB)
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

		// Check if Monitor StatefulSet already exists, if not create a new one
		stsM = &appsv1.StatefulSet{}
		stsMName := getStatefulsetName(eventbroker.Name, "m")
		err = r.Get(ctx, types.NamespacedName{Name: stsMName, Namespace: eventbroker.Namespace}, stsM)
		if err != nil && errors.IsNotFound(err) {
			// Define a new statefulset
			stsM := r.createStatefulsetForEventBroker(stsMName, eventbroker)
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
			if stsM.Spec.Template.ObjectMeta.Annotations[dependenciesSignatureAnnotationName] != hash(eventbroker.Spec) {
				// If resource versions differ it means update is required
				log.Info("Updating existing Monitor StatefulSet", " StatefulSet.Name", stsM.Name)
				r.updateStatefulsetForEventBroker(stsMName, eventbroker, stsM)
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

	// Check if pods are out-of-sync and need to be restarted
	// First check for readiness of all broker nodes to continue
	// TODO: emit events here instead of logs
	if stsP.Status.ReadyReplicas < 1 {
		log.Info("Detected unready Primary StatefulSet, waiting to be ready")
		return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
	} else if eventbroker.Spec.Redundancy {
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
	// // Next restart any pods that don't have the statefulset's lastAppliedConfig
	// // Must distinguish between HA and non-HA
	// if haDeployment {
	// 	// The algorithm is to process the Monitor, then the pod with `active=false`, finally `active=true`
	// } else {
	// 	// Kill the single pod if needed
		
	// }

	// Update the EventBroker status with the pod names
	// List the pods for this eventbroker's StatefulSet
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(eventbroker.Namespace),
		client.MatchingLabels(baseLabels(eventbroker.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "EventBroker.Namespace", eventbroker.Namespace, "EventBroker.Name", eventbroker.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)
	// Update status.BrokerPods if needed
	if !reflect.DeepEqual(podNames, eventbroker.Status.BrokerPods) {
		eventbroker.Status.BrokerPods = podNames
		err := r.Status().Update(ctx, eventbroker)
		if err != nil {
			log.Error(err, "Failed to update EventBroker status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// baseLabels returns the labels for selecting the resources
// belonging to the given eventbroker CR name.
func baseLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance": name,
		"app.kubernetes.io/name":     "eventbroker",
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
func (r *EventBrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventbrokerv1alpha1.EventBroker{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
