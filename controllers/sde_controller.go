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
	_ "embed"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	sdev1beta1 "sde.domain/sdeController/api/v1beta1"
)

// SdeReconciler reconciles a Sde object
type SdeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//go:embed embeds/db_cleanup.sh
var dbCleanup string

//+kubebuilder:rbac:groups=sde.sde.domain,resources=sdes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sde.sde.domain,resources=sdes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sde.sde.domain,resources=sdes/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *SdeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctxlog := log.FromContext(ctx)
	sde := &sdev1beta1.Sde{}
	err := r.Get(ctx, req.NamespacedName, sde)
	if err != nil {
		ctxlog.Error(err, "Operator not found")
		return ctrl.Result{}, err
	}

	// Reconcile DB
	if err = r.reconcileDb(ctx, sde); err != nil {
		ctxlog.Error(err, "PG Cleanup failed")
		return ctrl.Result{}, err
	}

	ctxlog.Info("All done")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SdeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sdev1beta1.Sde{}).
		Complete(r)
}
