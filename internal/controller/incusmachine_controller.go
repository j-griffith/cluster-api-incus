/*
Copyright 2026.

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

package controller

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	infrastructurev1alpha1 "github.com/j-griffith/cluster-api-provider-incus/api/v1alpha1"
	"github.com/j-griffith/cluster-api-provider-incus/internal/incus"
)

const incusMachineFinalizer = "infrastructure.cluster.x-k8s.io/incusmachine"

// IncusMachineReconciler reconciles a IncusMachine object
type IncusMachineReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	IncusClient incus.Client
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=incusmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=incusmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=incusmachines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IncusMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	incusMachine := &infrastructurev1alpha1.IncusMachine{}
	if err := r.Get(ctx, req.NamespacedName, incusMachine); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !incusMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, incusMachine)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(incusMachine, incusMachineFinalizer) {
		controllerutil.AddFinalizer(incusMachine, incusMachineFinalizer)
		if err := r.Update(ctx, incusMachine); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return r.reconcileNormal(ctx, log, incusMachine)
}

func (r *IncusMachineReconciler) reconcileNormal(ctx context.Context, log logr.Logger, incusMachine *infrastructurev1alpha1.IncusMachine) (ctrl.Result, error) {
	instanceName := incusMachine.Name
	if incusMachine.Status.InstanceID != "" {
		instanceName = incusMachine.Status.InstanceID
	}

	// Check if instance already exists
	exists, err := r.IncusClient.InstanceExists(ctx, instanceName)
	if err != nil {
		log.Error(err, "Failed to check if instance exists")
		return ctrl.Result{}, err
	}

	if exists {
		// Instance already created, ensure status is updated
		if incusMachine.Status.InstanceID != instanceName {
			incusMachine.Status.InstanceID = instanceName
			if err := r.Status().Update(ctx, incusMachine); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Create the VM instance
	image := incusMachine.Spec.Image
	if image == "" {
		image = "images:ubuntu/24.04"
	}
	cpus := incusMachine.Spec.CPUs
	if cpus < 1 {
		cpus = 2
	}
	memoryMiB := incusMachine.Spec.MemoryMiB
	if memoryMiB < 1 {
		memoryMiB = 2048
	}
	rootDiskSizeGiB := incusMachine.Spec.RootDiskSizeGiB

	if err := r.IncusClient.CreateInstance(ctx, instanceName, image, cpus, memoryMiB, rootDiskSizeGiB); err != nil {
		log.Error(err, "Failed to create Incus instance")
		return ctrl.Result{}, err
	}

	incusMachine.Status.InstanceID = instanceName
	if err := r.Status().Update(ctx, incusMachine); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Created Incus VM instance", "instance", instanceName)
	return ctrl.Result{}, nil
}

func (r *IncusMachineReconciler) reconcileDelete(ctx context.Context, log logr.Logger, incusMachine *infrastructurev1alpha1.IncusMachine) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(incusMachine, incusMachineFinalizer) {
		return ctrl.Result{}, nil
	}

	instanceName := incusMachine.Status.InstanceID
	if instanceName == "" {
		instanceName = incusMachine.Name
	}

	if instanceName != "" {
		exists, err := r.IncusClient.InstanceExists(ctx, instanceName)
		if err != nil {
			log.Error(err, "Failed to check if instance exists during deletion")
			return ctrl.Result{}, err
		}

		if exists {
			if err := r.IncusClient.DeleteInstance(ctx, instanceName); err != nil {
				log.Error(err, "Failed to delete Incus instance")
				return ctrl.Result{}, err
			}
			log.Info("Deleted Incus VM instance", "instance", instanceName)
		}
	}

	controllerutil.RemoveFinalizer(incusMachine, incusMachineFinalizer)
	if err := r.Update(ctx, incusMachine); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IncusMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha1.IncusMachine{}).
		Named("incusmachine").
		Complete(r)
}
