package devfile

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateBuildPod(pvcName string, container corev1.Container, cheProjectsRoot string) *corev1.Pod {
	// TODO(tkral): command should be something properly handles SIGTERM signal
	container.Command = []string{
		"sleep",
		"1h",
	}
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      pvcName,
			MountPath: cheProjectsRoot,
		})

	container.Name = "build"
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "CHE_PROJECTS_ROOT",
		Value: cheProjectsRoot,
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

func GenerateRunPod(pvcName string, container corev1.Container, command string, workingDir string, cheProjectsRoot string) *corev1.Pod {
	container.Command = []string{
		"sh", "-c",
		fmt.Sprintf("cd %s && %s", workingDir, command),
	}
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      pvcName,
			MountPath: cheProjectsRoot,
		})

	container.Name = "build"
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "CHE_PROJECTS_ROOT",
		Value: cheProjectsRoot,
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

func GenerateRunDeployment(pvcName string, container corev1.Container, command string, workingDir string, cheProjectsRoot string) *appsv1.Deployment {

	// overwrite and add few stuff to the container
	container.Name = "run"
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "CHE_PROJECTS_ROOT",
		Value: cheProjectsRoot,
	})
	container.Command = []string{
		"sh", "-c",
		fmt.Sprintf("cd %s && %s", workingDir, command),
	}
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      pvcName,
			MountPath: cheProjectsRoot,
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

func GenerateFatDeployment(deploymentName string, pvcName string, cheProjectsRoot string, devf Devfile, buildCommandName string, runCommandName string) (*appsv1.Deployment, error) {
	supervisordVolumeName := "supervisord-volume"

	// construct build container
	buildAction, err := devf.GetCommandAction(buildCommandName)
	if err != nil {
		return nil, err
	}
	devfBuildComponent, err := devf.GetComponent(*buildAction.Component)
	if err != nil {
		return nil, err
	}
	buildContainer, err := devfBuildComponent.ConvertToContainer()
	if err != nil {
		return nil, err
	}
	buildContainer.Name = "build"
	buildContainer.Env = append(buildContainer.Env, corev1.EnvVar{
		Name:  "CHE_PROJECTS_ROOT",
		Value: cheProjectsRoot,
	})
	// TODO(tkra): command should be something that has infinite sleep and has proper signal handling (SIGTERM)
	buildContainer.Command = []string{
		"/opt/odo/bin/go-init",
	}
	buildContainer.Args = []string{
		"-main",
		"sleep 3600",
	}
	buildContainer.VolumeMounts = append(buildContainer.VolumeMounts,
		corev1.VolumeMount{
			Name:      pvcName,
			MountPath: cheProjectsRoot,
		})
	buildContainer.VolumeMounts = append(buildContainer.VolumeMounts,
		corev1.VolumeMount{
			Name:      supervisordVolumeName,
			MountPath: "/opt/odo/",
		})

	// get info about run action
	runAction, err := devf.GetCommandAction(runCommandName)
	if err != nil {
		return nil, err
	}
	devfRunComponent, err := devf.GetComponent(*runAction.Component)
	if err != nil {
		return nil, err
	}
	runContainer, err := devfRunComponent.ConvertToContainer()
	if err != nil {
		return nil, err
	}
	runContainer.Name = "run"
	runContainer.Env = append(runContainer.Env, corev1.EnvVar{
		Name:  "CHE_PROJECTS_ROOT",
		Value: cheProjectsRoot,
	})
	runContainer.Env = append(runContainer.Env, corev1.EnvVar{
		Name:  "DEVFILE_RUN_COMMAND",
		Value: *runAction.Command,
	})
	runContainer.Env = append(runContainer.Env, corev1.EnvVar{
		Name:  "DEVFILE_RUN_WORKDIR",
		Value: *runAction.Workdir,
	})
	// TODO(tkral): command should be something that has infinite sleep and has proper signal handling (SIGTERM)
	runContainer.Command = []string{
		"/opt/odo/bin/go-init",
	}
	runContainer.Args = []string{
		"-main",
		"/opt/odo/bin/supervisord -c /opt/odo/conf/supervisor-devfile.conf",
	}
	runContainer.VolumeMounts = append(runContainer.VolumeMounts,
		corev1.VolumeMount{
			Name:      pvcName,
			MountPath: cheProjectsRoot,
		})
	runContainer.VolumeMounts = append(runContainer.VolumeMounts,
		corev1.VolumeMount{
			Name:      supervisordVolumeName,
			MountPath: "/opt/odo/",
		})

	initContainer := corev1.Container{
		Name:            "copy-supervisord",
		Image:           "quay.io/tkral/odo-supervisord-image:devfile-poc",
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      supervisordVolumeName,
				MountPath: "/opt/odo/",
			},
		},
		Command: []string{
			"/usr/bin/cp",
		},
		Args: []string{
			"-r",
			"/opt/odo-init/.",
			"/opt/odo/",
		},
	}

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{
			*runContainer,
			*buildContainer,
		},
		InitContainers: []corev1.Container{
			initContainer,
		},
		Volumes: []corev1.Volume{
			{
				Name: pvcName,
				// VolumeSource: corev1.VolumeSource{
				// 	PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				// 		ClaimName: pvcName,
				// 	},
				// },
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: supervisordVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
	}

	replicas := int32(1)
	dc := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentName,
			Labels: map[string]string{
				"devfile.odo.openshift.io": deploymentName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"devfile.odo.openshift.io": deploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"devfile.odo.openshift.io": deploymentName,
					},
				},
				Spec: podSpec,
			},
		},
	}
	return &dc, nil
}
