package component

import (
	"bufio"
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
	projectFilesBase := "/projects"
	projectFilesPath := fmt.Sprintf("%s/%s", projectFilesBase, devf.Projects[0].Name)
	buildCommand := "maven build"
	runCommand := "run webapp"

	buildAction, err := devf.GetCommandAction(buildCommand)
	if err != nil {
		return err
	}
	runAction, err := devf.GetCommandAction(runCommand)
	if err != nil {
		return err
	}

	localPath := "./"

	// TODO(tkral): remove this
	odoDir := filepath.Join(localPath, ".odo")
	if _, err := os.Stat(odoDir); os.IsNotExist(err) {
		glog.V(4).Infof("Creating directory %s", odoDir)
		errMkdir := os.Mkdir(odoDir, 0755)
		if errMkdir != nil {
			return errMkdir
		}
	}

	client, err := occlient.New()
	if err != nil {
		return err
	}

	// bootstrap files into the PVC

	// create pvc and pod only if it already doesn't exist
	//TODO(tkral): temporary hacky way
	// it should also check that it is in running state
	buildPod, err := client.GetOnePodFromSelector("podkind.odo.openshfit.io=build")
	if err != nil {
		_, err = client.CreatePVC(projectFilesPVC, "1Gi", map[string]string{})
		if err != nil {
			return err
		}

		devfBuildComponent, err := devf.GetComponent(*buildAction.Component)
		if err != nil {
			return err
		}
		buildContainer, err := devfBuildComponent.ConvertToContainer()
		if err != nil {
			return err
		}
		pod := devfile.GenerateBuildPod(projectFilesPVC, *buildContainer)
		_, err = client.CreatePod(pod)
		if err != nil {
			return err
		}
	}

	filesChanged, filesDeleted, err := util.RunIndexer(localPath, []string{})
	if err != nil {
		return err
	}
	glog.V(5).Infof("Indexer results: ")
	glog.V(5).Infof("filesChanged = %v", filesChanged)
	glog.V(5).Infof("filesDeleted = %v", filesDeleted)

	// use build container to sync files to volume
	err = client.SyncFiles("podkind.odo.openshfit.io=build", localPath, projectFilesPath, filesChanged, filesDeleted, pdo.forcePush, []string{})
	if err != nil {
		return err
	}

	s := log.SpinnerNoSpin("Building component")

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
	err = client.ExecCMDInContainer(buildPod.Name,
		[]string{"sh", "-c", fmt.Sprintf("cd %s && %s", *buildAction.Workdir, *buildAction.Command)},
		pipeWriter, pipeWriter, nil, false)

	if err != nil {
		// If we fail, log the output
		log.Errorf("Unable to build files\n%v", cmdOutput)
		s.End(false)
		return err
	}
	s.End(true)

	runPod, err := client.GetOnePodFromSelector("podkind.odo.openshfit.io=run")
	if err == nil {
		// if pod exist delete it
		errDelete := client.DeletePod(runPod.Name)
		if errDelete != nil {
			return errDelete
		}
	}
	// TODO(tkral): wait for pod to be deleted
	// or use Deployment

	devfRunComponent, err := devf.GetComponent(*runAction.Component)
	if err != nil {
		return err
	}
	runContainer, err := devfRunComponent.ConvertToContainer()
	if err != nil {
		return err
	}
	pod := devfile.GenerateRunPod(projectFilesPVC, *runContainer, *runAction.Command, *runAction.Workdir)
	_, err = client.CreatePod(pod)
	if err != nil {
		return err
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
