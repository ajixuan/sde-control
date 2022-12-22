package controllers

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	sdev1beta1 "sde.domain/sdeController/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *SdeReconciler) MakeJob(ctx context.Context, sde *sdev1beta1.Sde) (ctrl.Result, error) {
	ctxlog := log.FromContext(ctx)

	// STEP 2: create the ConfigMap with the script's content.
	configmap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: "run-scripts", Namespace: sde.Namespace}, configmap)
	if err != nil && errors.IsNotFound(err) {
		ctxlog.Error(err, "reason why not found")
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
						RestartPolicy: corev1.RestartPolicyOnFailure,
						// STEP 3a: define the ConfigMap as a volume.
						Volumes: []corev1.Volume{
							{
								Name: "task-script-volume",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "run-scripts",
										},
										DefaultMode: &configmapMode,
									},
								},
							},
							{
								Name: "db-secret-volume",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: fmt.Sprintf("%s-database-secrets", sde.Namespace),
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "task",
								Image: "postgres:12",
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
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "task-script-volume",
										MountPath: "/scripts",
										ReadOnly:  true,
									},
									{
										Name:      "db-secret-volume",
										MountPath: "/secrets",
										ReadOnly:  true,
									},
								},
								// STEP 3c: run the volume-mounted script.
								Command: []string{"/scripts/db_cleanup.sh", "postgres", "/secrets/ADMIN_DATABASE_PASSWORD", "", "sde_", "1"},
								// Command: []string{"sleep", "1231273"},
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
	}

	// Requeue if the job is not complete.
	if *job.Spec.Completions == 0 {
		ctxlog.Info("Requeuing to wait for Job to complete")
		return ctrl.Result{RequeueAfter: time.Second * 15}, nil
	}

	return ctrl.Result{}, nil
}
