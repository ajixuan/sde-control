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
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Sde object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *SdeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	ctxlog := log.FromContext(ctx)
	sde := &sdev1beta1.Sde{}
	r.Get(ctx, req.NamespacedName, sde)

	// STEP 2: create the ConfigMap with the script's content.
	configmap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: "run-scripts", Namespace: sde.Namespace}, configmap)
	if err != nil && errors.IsNotFound(err) {

		ctxlog.Info("Creating new ConfigMap")
		configmap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run-scripts",
				Namespace: sde.Namespace,
			},
			Data: map[string]string{
				"db_cleanup.sh": dbCleanup,
			},
		}

		err = ctrl.SetControllerReference(sde, configmap, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = r.Create(ctx, configmap)
		if err != nil {
			ctxlog.Error(err, "Failed to create ConfigMap")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil

	}

	// STEP 3: create the Job with the ConfigMap attached as a volume.
	job := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: "sde-controller-job", Namespace: sde.Namespace}, job)
	if err != nil && errors.IsNotFound(err) {

		ctxlog.Info("Creating new Job")
		configmapMode := int32(0554)
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sde-controller-job",
				Namespace: sde.Namespace,
			},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						// STEP 3a: define the ConfigMap as a volume.
						Volumes: []corev1.Volume{{
							Name: "task-script-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "run-scripts",
									},
									DefaultMode: &configmapMode,
								},
							},
						}},
						Containers: []corev1.Container{
							{
								Name:  "task",
								Image: "busybox",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(50), resource.DecimalSI),
										corev1.ResourceMemory: *resource.NewScaledQuantity(int64(250), resource.Mega),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(100), resource.DecimalSI),
										corev1.ResourceMemory: *resource.NewScaledQuantity(int64(500), resource.Mega),
									},
								},
								// STEP 3b: mount the ConfigMap volume.
								VolumeMounts: []corev1.VolumeMount{{
									Name:      "task-script-volume",
									MountPath: "/scripts",
									ReadOnly:  true,
								}},
								// STEP 3c: run the volume-mounted script.
								Command: []string{"/scripts/db_cleanup.sh"},
							},
						},
					},
				},
			},
		}

		err = ctrl.SetControllerReference(sde, job, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = r.Create(ctx, job)
		if err != nil {
			ctxlog.Error(err, "Failed to create Job")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Requeue if the job is not complete.
	if *job.Spec.Completions == 0 {
		ctxlog.Info("Requeuing to wait for Job to complete")
		return ctrl.Result{RequeueAfter: time.Second * 15}, nil
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
