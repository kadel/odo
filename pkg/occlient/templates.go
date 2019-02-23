package occlient

import (
	"fmt"
	"strings"

	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	componentlabels "github.com/redhat-developer/odo/pkg/component/labels"
	"github.com/redhat-developer/odo/pkg/config"
	"github.com/redhat-developer/odo/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CommonImageMeta has all the most common image data that is passed around within Odo
type CommonImageMeta struct {
	Name      string
	Tag       string
	Namespace string
	Ports     []corev1.ContainerPort
}

func applyConfigToDeploymentConfig(componentConfig config.ComponentSettings, dc *appsv1.DeploymentConfig, namespacedDCName string, appName string) {

	componentName := ""
	if componentConfig.ComponentName != nil {
		componentName = *(componentConfig.ComponentName)
	} else {
		// extract component name from dc
		componentName = dc.ObjectMeta.Labels[componentlabels.ComponentLabel]
	}

	// Get Image params from existing dc
	// ToDo: Add image details to config
	triggers := dc.Spec.Triggers
	imageNS := ""
	imageName := ""
	for _, trigger := range triggers {
		if trigger.Type == "ImageChange" {
			imageNS = trigger.ImageChangeParams.From.Namespace
			imageName = trigger.ImageChangeParams.From.Name
		}
	}

	componentType := ""
	if componentConfig.ComponentType != nil {
		componentType = *(componentConfig.ComponentType)
	} else {
		componentType = strings.Split(imageName, ":")[0]
	}

	// Retrieve labels
	// Save component type as label
	labels := componentlabels.GetLabels(componentName, appName, true)
	labels[componentlabels.ComponentTypeLabel] = componentType
	// ToDo(@anmolbabu): Add logic to persist and here, fetch component version
	labels[componentlabels.ComponentTypeVersion] = dc.ObjectMeta.Labels[componentlabels.ComponentTypeVersion] //imageTag

	// ObjectMetadata are the same for all generated objects
	// Create common metadata that will be updated throughout all objects.
	commonObjectMeta := metav1.ObjectMeta{
		Name:   namespacedDCName,
		Labels: labels,
		// ToDo(@anmolbabu): Create annotations
		Annotations: dc.ObjectMeta.Annotations,
	}

	// Gather the common image data into one struct
	commonImageMeta := CommonImageMeta{
		Name:      labels[componentlabels.ComponentTypeLabel],
		Tag:       labels[componentlabels.ComponentTypeVersion],
		Namespace: imageNS,
		Ports:     dc.Spec.Template.Spec.Containers[0].Ports,
	}

	resourceReqs := []util.ResourceRequirementInfo{}
	var resourceRequirements *corev1.ResourceRequirements
	if componentConfig.MinCPU != nil && componentConfig.MaxCPU != nil {
		cpuResourceConstraints := util.FetchResourceQuantity(corev1.ResourceCPU, *componentConfig.MinCPU, *componentConfig.MaxCPU, "")
		resourceReqs = append(resourceReqs, *cpuResourceConstraints)
	}
	if componentConfig.MinMemory != nil && componentConfig.MaxMemory != nil {
		memoryResourceConstraints := util.FetchResourceQuantity(corev1.ResourceMemory, *componentConfig.MinMemory, *componentConfig.MaxMemory, "")
		resourceReqs = append(resourceReqs, *memoryResourceConstraints)
	}
	if len(resourceReqs) > 0 {
		resourceRequirements = getResourceRequirementsFromRawData(resourceReqs)
	}

	*dc = generateSupervisordDeploymentConfig(
		commonObjectMeta,
		fmt.Sprintf("%s:%s", labels[componentlabels.ComponentTypeLabel], labels[componentlabels.ComponentTypeVersion]),
		commonImageMeta,
		dc.Spec.Template.Spec.Containers[0].Env,
		dc.Spec.Template.Spec.Containers[0].EnvFrom,
		resourceRequirements,
	)

	// Add the appropriate bootstrap volumes for SupervisorD
	addBootstrapVolumeCopyInitContainer(dc, commonObjectMeta.Name)
	addBootstrapSupervisordInitContainer(dc, commonObjectMeta.Name)
}

func generateSupervisordDeploymentConfig(commonObjectMeta metav1.ObjectMeta, builderImage string, commonImageMeta CommonImageMeta,
	envVar []corev1.EnvVar, envFrom []corev1.EnvFromSource, resourceRequirements *corev1.ResourceRequirements) appsv1.DeploymentConfig {

	// Generates and deploys a DeploymentConfig with an InitContainer to copy over the SupervisorD binary.
	dc := appsv1.DeploymentConfig{
		ObjectMeta: commonObjectMeta,
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.DeploymentStrategyTypeRecreate,
			},
			Selector: map[string]string{
				"deploymentconfig": commonObjectMeta.Name,
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deploymentconfig": commonObjectMeta.Name,
					},
					// https://github.com/redhat-developer/odo/pull/622#issuecomment-413410736
					Annotations: map[string]string{
						"alpha.image.policy.openshift.io/resolve-names": "*",
					},
				},
				Spec: corev1.PodSpec{
					// The application container
					Containers: []corev1.Container{
						{
							Image: builderImage,
							Name:  commonObjectMeta.Name,
							Ports: commonImageMeta.Ports,
							// Run the actual supervisord binary that has been mounted into the container
							Command: []string{
								"/var/lib/supervisord/bin/dumb-init",
								"--",
							},
							// Using the appropriate configuration file that contains the "run" script for the component.
							// either from: /usr/libexec/s2i/assemble or /opt/app-root/src/.s2i/bin/assemble
							Args: []string{
								"/var/lib/supervisord/bin/supervisord",
								"-c",
								"/var/lib/supervisord/conf/supervisor.conf",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      supervisordVolumeName,
									MountPath: "/var/lib/supervisord",
								},
							},
							Env:     envVar,
							EnvFrom: envFrom,
						},
					},

					// Create a volume that will be shared betwen InitContainer and the applicationContainer
					// in order to pass over the SupervisorD binary
					Volumes: []corev1.Volume{
						{
							Name: supervisordVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
			// We provide triggers to create an ImageStream so that the application container will use the
			// correct and approriate image that's located internally within the OpenShift commonObjectMeta.Namespace
			Triggers: []appsv1.DeploymentTriggerPolicy{
				{
					Type: "ConfigChange",
				},
				{
					Type: "ImageChange",
					ImageChangeParams: &appsv1.DeploymentTriggerImageChangeParams{
						Automatic: true,
						ContainerNames: []string{
							commonObjectMeta.Name,
							"copy-files-to-volume",
						},
						From: corev1.ObjectReference{
							Kind:      "ImageStreamTag",
							Name:      fmt.Sprintf("%s:%s", commonImageMeta.Name, commonImageMeta.Tag),
							Namespace: commonImageMeta.Namespace,
						},
					},
				},
			},
		},
	}
	containerIndex := -1
	if resourceRequirements != nil {
		for index, container := range dc.Spec.Template.Spec.Containers {
			if container.Name == commonObjectMeta.Name {
				containerIndex = index
				break
			}
		}
		if containerIndex != -1 {
			dc.Spec.Template.Spec.Containers[containerIndex].Resources = *resourceRequirements
		}
	}
	return dc
}

func fetchContainerResourceLimits(container corev1.Container) corev1.ResourceRequirements {
	return container.Resources
}

func getResourceRequirementsFromRawData(resources []util.ResourceRequirementInfo) *corev1.ResourceRequirements {
	if len(resources) == 0 {
		return nil
	}
	var resourceRequirements corev1.ResourceRequirements
	for _, resource := range resources {
		if resourceRequirements.Limits == nil {
			resourceRequirements.Limits = make(corev1.ResourceList)
		}
		if resourceRequirements.Requests == nil {
			resourceRequirements.Requests = make(corev1.ResourceList)
		}
		resourceRequirements.Limits[resource.ResourceType] = resource.MaxQty
		resourceRequirements.Requests[resource.ResourceType] = resource.MinQty
	}
	return &resourceRequirements
}

func generateGitDeploymentConfig(commonObjectMeta metav1.ObjectMeta, image string, containerPorts []corev1.ContainerPort, envVars []corev1.EnvVar, resourceRequirements *corev1.ResourceRequirements) appsv1.DeploymentConfig {
	dc := appsv1.DeploymentConfig{
		ObjectMeta: commonObjectMeta,
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.DeploymentStrategyTypeRecreate,
			},
			Selector: map[string]string{
				"deploymentconfig": commonObjectMeta.Name,
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deploymentconfig": commonObjectMeta.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: image,
							Name:  commonObjectMeta.Name,
							Ports: containerPorts,
							Env:   envVars,
						},
					},
				},
			},
			Triggers: []appsv1.DeploymentTriggerPolicy{
				{
					Type: "ConfigChange",
				},
				{
					Type: "ImageChange",
					ImageChangeParams: &appsv1.DeploymentTriggerImageChangeParams{
						Automatic: true,
						ContainerNames: []string{
							commonObjectMeta.Name,
						},
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: image,
						},
					},
				},
			},
		},
	}
	containerIndex := -1
	if resourceRequirements != nil {
		for index, container := range dc.Spec.Template.Spec.Containers {
			if container.Name == commonObjectMeta.Name {
				containerIndex = index
				break
			}
		}
		if containerIndex != -1 {
			dc.Spec.Template.Spec.Containers[containerIndex].Resources = *resourceRequirements
		}
	}
	return dc
}

// generateBuildConfig creates a BuildConfig for Git URL's being passed into Odo
func generateBuildConfig(commonObjectMeta metav1.ObjectMeta, gitURL, gitRef, imageName, imageNamespace string) buildv1.BuildConfig {

	buildSource := buildv1.BuildSource{
		Git: &buildv1.GitBuildSource{
			URI: gitURL,
			Ref: gitRef,
		},
		Type: buildv1.BuildSourceGit,
	}

	return buildv1.BuildConfig{
		ObjectMeta: commonObjectMeta,
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: commonObjectMeta.Name + ":latest",
					},
				},
				Source: buildSource,
				Strategy: buildv1.BuildStrategy{
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: corev1.ObjectReference{
							Kind:      "ImageStreamTag",
							Name:      imageName,
							Namespace: imageNamespace,
						},
					},
				},
			},
		},
	}
}

//
// Below is related to SUPERVISORD
//

// AddBootstrapInitContainer adds the bootstrap init container to the deployment config
// dc is the deployment config to be updated
// dcName is the name of the deployment config
func addBootstrapVolumeCopyInitContainer(dc *appsv1.DeploymentConfig, dcName string) {
	dc.Spec.Template.Spec.InitContainers = append(dc.Spec.Template.Spec.InitContainers,
		corev1.Container{
			Name: "copy-files-to-volume",
			// Using custom image from bootstrapperImage variable for the initial initContainer
			Image: dc.Spec.Template.Spec.Containers[0].Image,
			Command: []string{
				"sh",
				"-c"},
			// Script required to copy over file information from /opt/app-root
			// Source https://github.com/jupyter-on-openshift/jupyter-notebooks/blob/master/minimal-notebook/setup-volume.sh
			Args: []string{`
SRC=/opt/app-root
DEST=/mnt/app-root

if [ -f $DEST/.delete-volume ]; then
    rm -rf $DEST
fi
 if [ -d $DEST ]; then
    if [ -f $DEST/.sync-volume ]; then
        if ! [[ "$JUPYTER_SYNC_VOLUME" =~ ^(false|no|n|0)$ ]]; then
            JUPYTER_SYNC_VOLUME=yes
        fi
    fi
     if [[ "$JUPYTER_SYNC_VOLUME" =~ ^(true|yes|y|1)$ ]]; then
        rsync -ar --ignore-existing $SRC/. $DEST
    fi
     exit
fi
 if [ -d $DEST.setup-volume ]; then
    rm -rf $DEST.setup-volume
fi

mkdir -p $DEST.setup-volume
tar -C $SRC -cf - . | tar -C $DEST.setup-volume -xvf -
mv $DEST.setup-volume $DEST
			`},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      getAppRootVolumeName(dcName),
					MountPath: "/mnt",
				},
			},
		})
}

// addBootstrapSupervisordInitContainer creates an init container that will copy over
// supervisord to the application image during the start-up procress.
func addBootstrapSupervisordInitContainer(dc *appsv1.DeploymentConfig, dcName string) {

	dc.Spec.Template.Spec.InitContainers = append(dc.Spec.Template.Spec.InitContainers,
		corev1.Container{
			Name:  "copy-supervisord",
			Image: getBootstrapperImage(),
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      supervisordVolumeName,
					MountPath: "/var/lib/supervisord",
				},
			},
			Command: []string{
				"/usr/bin/cp",
			},
			Args: []string{
				"-r",
				"/opt/supervisord",
				"/var/lib/",
			},
		})
}

// addBootstrapVolume adds the bootstrap volume to the deployment config
// dc is the deployment config to be updated
// dcName is the name of the deployment config
func addBootstrapVolume(dc *appsv1.DeploymentConfig, dcName string) {
	dc.Spec.Template.Spec.Volumes = append(dc.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: getAppRootVolumeName(dcName),
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: getAppRootVolumeName(dcName),
			},
		},
	})
}

// addBootstrapVolumeMount mounts the bootstrap volume to the deployment config
// dc is the deployment config to be updated
// dcName is the name of the deployment config
func addBootstrapVolumeMount(dc *appsv1.DeploymentConfig, dcName string) {
	for i := range dc.Spec.Template.Spec.Containers {
		dc.Spec.Template.Spec.Containers[i].VolumeMounts = append(dc.Spec.Template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      getAppRootVolumeName(dcName),
			MountPath: "/opt/app-root",
			SubPath:   "app-root",
		})
	}
}
