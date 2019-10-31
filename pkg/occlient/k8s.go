package occlient

import corev1 "k8s.io/api/core/v1"

func (c *Client) CreateFileCopierPod(pvcName string, volumePath string) (*corev1.Pod, error) {
	pod := generateFileCopierPod(pvcName, volumePath)
	createdPod, err := c.kubeClient.CoreV1().Pods(c.Namespace).Create(&pod)
	if err != nil {
		return nil, err
	}
	return createdPod, nil
}
