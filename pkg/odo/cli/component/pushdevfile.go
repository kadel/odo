package component

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/devfile"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

var pushDevFile = ktemplates.Examples(`  # TODO
%[1]s
  `)

const PushDevfileRecommendedCommandName = "push-devfile"

// PushDevfileOptions encapsulates odo component push-devfile  options
type PushDevfileOptions struct {
	// path to devfile
	devfile string

	*genericclioptions.Context
}

// NewPushDevfileOptions returns new instance of PushDevfileOptions
func NewPushDevfileOptions() *PushDevfileOptions {
	return &PushDevfileOptions{}
}

// Complete completes  args
func (pdo *PushDevfileOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	pdo.Context = genericclioptions.NewContext(cmd)
	return
}

// Validate validates the  parameters
func (pdo *PushDevfileOptions) Validate() (err error) {
	return nil
}

// Run has the logic to perform the required actions as part of command
func (pdo *PushDevfileOptions) Run() (err error) {
	glog.V(4).Infof("component push-devfile")
	glog.V(4).Infof("devfile: %s", pdo.devfile)

	devf, err := devfile.Load(pdo.devfile)
	if err != nil {
		return err
	}

	// bootstrap files into the PVC

	client, err := occlient.New()
	if err != nil {
		return err
	}

	projectFilesPVCName := "project-files"
	projectFilesBase := "/projects"
	projectFilesPath := fmt.Sprintf("%s/%s", projectFilesBase, devf.Projects[0].Name)

	_, err = client.CreatePVC(projectFilesPVCName, "1Gi", map[string]string{})
	if err != nil {
		return err
	}

	pod, err := client.CreateFileCopierPod(projectFilesPVCName, projectFilesPath)
	if err != nil {
		return err
	}
	fmt.Printf("pod = %#v\n", pod)
	// create pod and sync files to PVC

	// for _, component := range devf.Components {
	// 	if component.Type == devfile.DevfileComponentsTypeDockerimage {
	// 		err, container := devfile.ComponentToContainer(&component)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		fmt.Printf("containter = %#v\n", *container)
	// 	}

	// }

	return nil
}

// NewCmdPushDevfile implements odo push-devfile  command
func NewCmdPushDevfile(name, fullName string) *cobra.Command {
	o := NewPushDevfileOptions()

	var pushDevfileCmd = &cobra.Command{
		Use:     name,
		Short:   "Push component form devfile.",
		Long:    "Push component form devfile.",
		Example: fmt.Sprintf(getExample, fullName),
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	pushDevfileCmd.Flags().StringVar(&o.devfile, "devfile", "./devfile.yaml", "Path to a devfile.yaml")

	project.AddProjectFlag(pushDevfileCmd)

	return pushDevfileCmd
}
