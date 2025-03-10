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
package nova

import (
	env "github.com/openstack-k8s-operators/lib-common/modules/common/env"
	novav1 "github.com/openstack-k8s-operators/nova-operator/api/v1beta1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func HostDiscoveryJob(
	instance *novav1.NovaCell,
	configName string,
	scriptName string,
	inputHash string,
	labels map[string]string,
) *batchv1.Job {
	args := []string{"-c", KollaServiceCommand}

	envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVars["KOLLA_BOOTSTRAP"] = env.SetValue("true")

	// This is stored in the Job so that if the input of the job changes
	// then it results in a new job hash and therefore lib-common will re-run
	// the job
	envVars["INPUT_HASH"] = env.SetValue(inputHash)

	env := env.MergeEnvs([]corev1.EnvVar{}, envVars)

	jobName := instance.Name + "-host-discover"

	volumes := []corev1.Volume{
		GetConfigVolume(configName),
		GetScriptVolume(scriptName),
	}
	volumeMounts := []corev1.VolumeMount{
		GetConfigVolumeMount(),
		GetScriptVolumeMount(),
		GetKollaConfigVolumeMount("host-discover"),
	}

	// add CA cert if defined
	if instance.Spec.TLS.CaBundleSecretName != "" {
		volumes = append(volumes, instance.Spec.TLS.CreateVolume())
		volumeMounts = append(volumeMounts, instance.Spec.TLS.CreateVolumeMounts(nil)...)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: instance.Spec.ServiceAccount,
					Volumes:            volumes,
					Containers: []corev1.Container{
						{
							Name: "nova-manage",
							Command: []string{
								"/bin/bash",
							},
							Args:  args,
							Image: instance.Spec.ConductorContainerImageURL,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: ptr.To(NovaUserID),
							},
							Env:          env,
							VolumeMounts: volumeMounts,
						},
					},
				},
			},
		},
	}

	if instance.Spec.NodeSelector != nil {
		job.Spec.Template.Spec.NodeSelector = *instance.Spec.NodeSelector
	}

	return job
}
