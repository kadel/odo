package devfile

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateBuildPod(pvcName string, container corev1.Container) *corev1.Pod {
	container.Command = []string{
		"sleep",
		"1h",
	}
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name: pvcName,
			// TODO(tkral): use const
			MountPath: "/projects",
		})

	container.Name = "build"
	container.Env = append(container.Env, corev1.EnvVar{
		Name: "CHE_PROJECTS_ROOT",
		// TODO(tkral): use const
		Value: "/projects",
	})

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			// TODO(tkral): this should be prefixed with project name
			Name: "build",
			Labels: map[string]string{
				// TODO(tkral): use const
				"podkind.odo.openshfit.io": "build",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				container,
			},
			Volumes: []corev1.Volume{
				{
					Name: pvcName,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}
	return &pod
}

func GenerateRunPod(pvcName string, container corev1.Container, command string, workingDir string) *corev1.Pod {
	container.Command = []string{
		"sh", "-c",
		fmt.Sprintf("cd %s && %s", workingDir, command),
	}
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name: pvcName,
			// TODO(tkral): use const
			MountPath: "/projects",
		})

	container.Name = "build"
	container.Env = append(container.Env, corev1.EnvVar{
		Name: "CHE_PROJECTS_ROOT",
		// TODO(tkral): use const
		Value: "/projects",
	})

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			// TODO(tkral): this should be prefixed with project name
			Name: "run",
			Labels: map[string]string{
				// TODO(tkral): use const
				"podkind.odo.openshfit.io": "run",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				container,
			},
			Volumes: []corev1.Volume{
				{
					Name: pvcName,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}
	return &pod
}

func GenerateRunDeployment(pvcName string, container corev1.Container, command string, workingDir string) *appsv1.Deployment {

	// overwrite and add few stuff to the container
	container.Name = "build"
	container.Env = append(container.Env, corev1.EnvVar{
		Name: "CHE_PROJECTS_ROOT",
		// TODO(tkral): use const
		Value: "/projects",
	})
	container.Command = []string{
		"sh", "-c",
		fmt.Sprintf("cd %s && %s", workingDir, command),
	}
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name: pvcName,
			// TODO(tkral): use const
			MountPath: "/projects",
		})

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{
			container,
		},
		Volumes: []corev1.Volume{
			{
				Name: pvcName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			},
		},
	}

	replicas := int32(1)
	dc := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			// TODO(tkral): this should be prefixed with project name
			Name: "run",
			Labels: map[string]string{
				// TODO(tkral): use const
				"podkind.odo.openshfit.io": "run",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					// TODO(tkral): this should be prefixed with project name
					"deploymentconfig": "run",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deploymentconfig":         "run",
						"podkind.odo.openshfit.io": "run",
					},
				},
				Spec: podSpec,
			},
		},
	}
	return &dc
}
