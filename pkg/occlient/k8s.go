package occlient

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift/odo/pkg/log"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (c *Client) CreatePod(pod *corev1.Pod) (*corev1.Pod, error) {
	createdPod, err := c.kubeClient.CoreV1().Pods(c.Namespace).Create(pod)
	if err != nil {
		return nil, err
	}
	return createdPod, nil
}

func (c *Client) SyncFiles(podName string, containerName string, path string, targetPath string, files []string, delFiles []string, forcePush bool, globExps []string) error {

	// Sync the files to the pod
	s := log.Spinner("Syncing files to the component")
	defer s.End(false)

	// If there are files identified as deleted, propagate them to the component pod
	if len(delFiles) > 0 {
		glog.V(4).Infof("propogating deletion of files %s to pod", strings.Join(delFiles, " "))
		/*
			Delete files observed by watch to have been deleted from each of s2i directories like:
				deployment dir: In interpreted runtimes like python, source is copied over to deployment dir so delete needs to happen here as well
				destination dir: This is the directory where s2i expects source to be copied for it be built and deployed
				working dir: Directory where, sources are copied over from deployment dir from where the s2i builds and deploys source.
							 Deletes need to happen here as well otherwise, even if the latest source is copied over, the stale source files remain
				source backup dir: Directory used for backing up source across multiple iterations of push and watch in component container
								   In case of python, s2i image moves sources from destination dir to workingdir which means sources are deleted from destination dir
								   So, during the subsequent watch pushing new diff to component pod, the source as a whole doesn't exist at destination dir and hence needs
								   to be backed up.
		*/
		err := c.PropagateDeletesToContainer(podName, containerName, delFiles, []string{targetPath})
		if err != nil {
			return errors.Wrapf(err, "unable to propagate file deletions %+v", delFiles)
		}
	}

	if !forcePush {
		if len(files) == 0 && len(delFiles) == 0 {
			// nothing to push
			s.End(true)
			return nil
		}
	}

	if forcePush || len(files) > 0 {
		glog.V(4).Infof("Copying files %s to pod", strings.Join(files, " "))
		err := c.CopyFileToContainer(path, podName, containerName, targetPath, files, globExps)
		if err != nil {
			s.End(false)
			return errors.Wrap(err, "unable push files to pod")
		}
	}
	s.End(true)

	return nil
}

func (c *Client) DeletePod(name string) error {
	return c.kubeClient.CoreV1().Pods(c.Namespace).Delete(name, nil)
}

func (c *Client) CreateDeployment(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	createdDeployment, err := c.kubeClient.AppsV1().Deployments(c.Namespace).Create(deployment)
	if err != nil {
		return nil, err
	}
	return createdDeployment, nil
}

func (c *Client) GetDeploymentFromName(name string) (*appsv1.Deployment, error) {
	glog.V(4).Infof("Getting Deployment: %s", name)
	deploymentConfig, err := c.kubeClient.AppsV1().Deployments(c.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if !strings.Contains(err.Error(), fmt.Sprintf(DEPLOYMENT_CONFIG_NOT_FOUND_ERROR_STR, name)) {
			return nil, errors.Wrapf(err, "unable to get DeploymentConfig %s", name)
		} else {
			return nil, DEPLOYMENT_CONFIG_NOT_FOUND
		}
	}
	return deploymentConfig, nil
}

func (c *Client) GetServiceFromName(serviceName string) (*corev1.Service, error) {
	return c.kubeClient.CoreV1().Services(c.Namespace).Get(serviceName, metav1.GetOptions{})
}

func (c *Client) CreateServiceForPorts(serviceName string, podSelector map[string]string, containerPorts []corev1.ContainerPort) (*corev1.Service, error) {
	// generate and create Service
	var svcPorts []corev1.ServicePort
	for _, containerPort := range containerPorts {
		svcPort := corev1.ServicePort{

			Name:       containerPort.Name,
			Port:       containerPort.ContainerPort,
			Protocol:   containerPort.Protocol,
			TargetPort: intstr.FromInt(int(containerPort.ContainerPort)),
		}
		svcPorts = append(svcPorts, svcPort)
	}
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
		},
		Spec: corev1.ServiceSpec{
			Ports:    svcPorts,
			Selector: podSelector,
		},
	}
	createdSvc, err := c.kubeClient.CoreV1().Services(c.Namespace).Create(&svc)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create Service for %s", serviceName)
	}
	return createdSvc, err
}
