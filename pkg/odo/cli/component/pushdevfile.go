package component

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/devfile"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/util"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

var pushDevFile = ktemplates.Examples(`  # TODO
%[1]s
  `)

const PushDevfileRecommendedCommandName = "push-devfile"

// PushDevfileOptions encapsulates odo component push-devfile  options
type PushDevfileOptions struct {
	// path to devfile
	devfile   string
	forcePush bool

	*genericclioptions.Context
}

// NewPushDevfileOptions returns new instance of PushDevfileOptions
func NewPushDevfileOptions() *PushDevfileOptions {
	return &PushDevfileOptions{}
}

// Complete completes  args
func (pdo *PushDevfileOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	// pdo.Context = genericclioptions.NewContext(cmd)
	return nil
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

	projectFilesPVC := "project-files"
	cheProjectsRoot := "/projects"
	localPath := "./"
	projectFilesPath := fmt.Sprintf("%s/%s", cheProjectsRoot, devf.Projects[0].Name)
	// name of command that that will be executed to build the app
	buildCommand := "maven build"
	// name of the command that will be used to run the app
	runCommand := "run webapp"

	// TODO(tkral): remove this
	// Make sure that .odo directory exists, as  it is needed by fileindexer
	odoDir := filepath.Join(localPath, ".odo")
	if _, err := os.Stat(odoDir); os.IsNotExist(err) {
		glog.V(4).Infof("Creating directory %s", odoDir)
		errMkdir := os.Mkdir(odoDir, 0755)
		if errMkdir != nil {
			return errMkdir
		}
	}

	buildAction, err := devf.GetCommandAction(buildCommand)
	if err != nil {
		return err
	}

	// runAction, err := devf.GetCommandAction(runCommand)
	// if err != nil {
	// 	return err
	// }

	client, err := occlient.New()
	if err != nil {
		return err
	}

	// bootstrap files into the PVC
	pvc, err := client.GetPVCFromName(projectFilesPVC)
	if err != nil {
		// if pv doesn't exist, create it
		pvc, err = client.CreatePVC(projectFilesPVC, "1Gi", map[string]string{})
		if err != nil {
			return err
		}
		// force push if new pvc is being created
		pdo.forcePush = true

	}
	// TODO(tkral): generate the name based on devfile project info
	deploymentName := "devfile"
	deployment, err := client.GetDeploymentFromName(deploymentName)
	if err != nil {
		deployment, err = devfile.GenerateFatDeployment(deploymentName, pvc.Name, cheProjectsRoot, *devf, buildCommand, runCommand)
		if err != nil {
			return err
		}
		_, err = client.CreateDeployment(deployment)
		if err != nil {
			return err
		}
		// force push when deployment config is being created
		pdo.forcePush = true
	}

	filesChanged, filesDeleted, err := util.RunIndexer(localPath, []string{})
	if err != nil {
		return err
	}
	glog.V(5).Infof("Indexer results: ")
	glog.V(5).Infof("filesChanged = %v", filesChanged)
	glog.V(5).Infof("filesDeleted = %v", filesDeleted)

	// Wait for Pod to be in running state otherwise we can't sync data to it.
	pod, err := client.WaitAndGetPod("devfile.odo.openshift.io="+deploymentName, corev1.PodRunning, "Waiting for component to start")
	if err != nil {
		return err
	}

	err = client.SyncFiles(pod.Name, "build", localPath, projectFilesPath, filesChanged, filesDeleted, pdo.forcePush, []string{})
	if err != nil {
		return err
	}
	s := log.SpinnerNoSpin("Building source code")

	// use pipes to write output from ExecCMDInContainer in yellow  to 'out' io.Writer
	pipeReader, pipeWriter := io.Pipe()
	out := os.Stdout
	var cmdOutput string

	// This Go routine will automatically pipe the output from ExecCMDInContainer to
	// our logger.
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			line := scanner.Text()
			_, err := fmt.Fprintln(out, line)
			if err != nil {
				log.Errorf("Unable to print to stdout: %v", err)
			}
			cmdOutput += fmt.Sprintln(line)
		}
	}()

	// TODO(tkral): check Workingdir and Command
	err = client.ExecCMDInContainer(pod.Name, "build",
		[]string{"sh", "-c", fmt.Sprintf("cd %s && %s", *buildAction.Workdir, *buildAction.Command)},
		pipeWriter, pipeWriter, nil, false)

	if err != nil {
		// If we fail, log the output
		log.Errorf("Unable to build the source code\n%v", cmdOutput)
		s.End(false)
		return err
	}
	s.End(true)

	var superStdout, superStderr bytes.Buffer
	reloadCmd := []string{"sh", "-c", "/opt/odo/bin/supervisord ctl stop run ; /opt/odo/bin/supervisord ctl start run"}
	glog.V(4).Infof("Reload command failed %v", reloadCmd)
	err = client.ExecCMDInContainer(pod.Name, "run", reloadCmd, &superStdout, &superStderr, nil, false)
	glog.V(4).Infof("stdout: %s", superStdout.String())
	glog.V(4).Infof("stderr: %s", superStdout.String())
	if err != nil {
		return err
	}

	// Create service if doesn't exist yet
	svcName := deployment.ObjectMeta.Name
	_, err = client.GetServiceFromName(svcName)
	if err != nil {

		_, err = client.CreateServiceForPorts(svcName, deployment.Spec.Template.Labels, deployment.Spec.Template.Spec.Containers[0].Ports)
		if err != nil {
			return err
		}
	}

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

	pushDevfileCmd.Flags().BoolVarP(&o.forcePush, "force", "f", false, "Push all changes")

	project.AddProjectFlag(pushDevfileCmd)

	return pushDevfileCmd
}
