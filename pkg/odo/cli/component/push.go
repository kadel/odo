package component

import (
	"fmt"
	"io"

	"github.com/redhat-developer/odo/pkg/config"
	"github.com/redhat-developer/odo/pkg/occlient"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/project"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"

	"github.com/pkg/errors"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"

	"net/url"
	"os"
	"runtime"

	"github.com/redhat-developer/odo/pkg/log"
	odoutil "github.com/redhat-developer/odo/pkg/odo/util"

	"github.com/fatih/color"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/redhat-developer/odo/pkg/util"

	"path/filepath"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var pushCmdExample = ktemplates.Examples(`  # Push source code to the current component
%[1]s

# Push data to the current component from the original source.
%[1]s

# Push source code in ~/mycode to component called my-component
%[1]s my-component --local ~/mycode
  `)

// PushRecommendedCommandName is the recommended push command name
const PushRecommendedCommandName = "push"

// PushOptions encapsulates options that push command uses
type PushOptions struct {
	ignores          []string
	local            string
	sourceType       string
	sourcePath       string
	localConfig      *config.LocalConfigInfo
	componentContext string
	*genericclioptions.Context
	client *occlient.Client
}

// NewPushOptions returns new instance of PushOptions
func NewPushOptions() *PushOptions {
	return &PushOptions{
		ignores:     []string{},
		localConfig: &config.LocalConfigInfo{},
	}
}

// Complete completes push args
func (po *PushOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	po.client = genericclioptions.Client(cmd)

	var contextFlag *string
	if len(po.componentContext) > 0 {
		contextFlag = &(po.componentContext)
	}

	conf, err := config.NewLocalConfigInfo(contextFlag)
	if err != nil {
		return errors.Wrap(err, "failed to fetch component config")
	}
	po.localConfig = conf

	if _, err = os.Stat(po.localConfig.Filename); err != nil {
		return errors.Wrapf(err, "failed trying to read config file in %s", po.localConfig.Filename)
	}

	po.sourceType = *(po.localConfig.ComponentSettings.SourceType)
	if po.sourceType == string(util.LOCAL) {
		if len(po.componentContext) != 0 {
			po.sourcePath = util.GenFileURL(po.componentContext, runtime.GOOS)
		} else {
			po.sourcePath, err = os.Getwd()
			if err != nil {
				return errors.Wrapf(err, "failed to create component with config %+v", po.localConfig)
			}
		}
	}
	if po.sourceType == string(util.BINARY) {
		po.sourcePath = *(po.localConfig.ComponentSettings.Path)
	}

	if po.sourceType == string(util.BINARY) || po.sourceType == string(util.LOCAL) {
		u, err := url.Parse(po.sourcePath)
		if err != nil {
			return errors.Wrapf(err, "unable to parse source %s from component %s", po.sourcePath, *(po.localConfig.ComponentSettings.ComponentName))
		}

		if u.Scheme != "" && u.Scheme != "file" {
			return fmt.Errorf("Component %s has invalid source path %s", *(po.localConfig.ComponentSettings.ComponentName), u.Scheme)
		}
		po.sourcePath = util.ReadFilePath(u, runtime.GOOS)
	}

	if len(po.ignores) == 0 {
		rules, err := util.GetIgnoreRulesFromDirectory(po.sourcePath)
		if err != nil {
			odoutil.LogErrorAndExit(err, "")
		}
		po.ignores = append(po.ignores, rules...)
	}
	po.Context = genericclioptions.NewContextCreatingAppIfNeeded(cmd)
	return
}

// Validate validates the push parameters
func (po *PushOptions) Validate() (err error) {
	// if the componentName is blank then there is no active component set
	if (po.localConfig.ComponentSettings.ComponentName == nil) && len(*(po.localConfig.ComponentSettings.ComponentName)) == 0 {
		return fmt.Errorf("no component is set in component config. Use 'odo config set ComponentName <component_name>'")
	}

	return
}

func (po *PushOptions) createCmpIfNotExists(stdout io.Writer) error {
	isPrjExists, err := project.Exists(po.client, *(po.localConfig.ComponentSettings.Project))
	if err != nil {
		return errors.Wrapf(err, "failed to check if project with name %s exists", *(po.localConfig.ComponentSettings.Project))
	}
	if !isPrjExists {
		log.Successf("Creating project %s", *(po.localConfig.ComponentSettings.Project))
		err = project.Create(po.client, *(po.localConfig.ComponentSettings.Project), true)
		if err != nil {
			log.Errorf("Failed creating project %s", *(po.localConfig.ComponentSettings.Project))
			return errors.Wrapf(
				err,
				"project %s does not exist. Failed creating it.Please try after creating project using `odo project create <project_name>`",
				*(po.localConfig.ComponentSettings.Project),
			)
		}
		log.Successf("Successfully created project %s", *(po.localConfig.ComponentSettings.Project))
	}
	po.client.Namespace = *(po.localConfig.ComponentSettings.Project)

	isCmpExists, err := component.Exists(po.client, *(po.localConfig.ComponentSettings.ComponentName), *(po.localConfig.ComponentSettings.App))
	if err != nil {
		return errors.Wrapf(err, "failed to check if component %s exists or not", *(po.localConfig.ComponentSettings.ComponentName))
	}

	if !isCmpExists {
		log.Successf("Creating %s component with name %s", *(po.localConfig.ComponentSettings.ComponentType), *(po.localConfig.ComponentSettings.ComponentName))
		// Classic case of component creation
		if err = component.CreateComponent(po.client, po.localConfig.ComponentSettings, po.componentContext, stdout); err != nil {
			log.Errorf("Failed to create component with settings %+v. Error: %+v", po.localConfig.ComponentSettings, err)
			os.Exit(1)
		}
		log.Successf("Successfully created component %s", *(po.localConfig.ComponentSettings.ComponentName))
	} else {
		log.Successf("Applying component settings %+v to component: %v", po.localConfig, *(po.localConfig.ComponentSettings.ComponentName))
		// Apply config
		err = component.ApplyConfig(po.client, po.localConfig.ComponentSettings, po.componentContext, stdout)
		if err != nil {
			log.Errorf("Failed to update config to component deployed. Error %+v", err)
			os.Exit(1)
		}
		log.Successf("Successfully applied component settings %+v to component: %v", po.localConfig.ComponentSettings, *(po.localConfig.ComponentSettings.ComponentName))
	}
	return nil
}

// Run has the logic to perform the required actions as part of command
func (po *PushOptions) Run() (err error) {
	stdout := color.Output

	err = po.createCmpIfNotExists(stdout)
	if err != nil {
		return
	}

	log.Successf("Pushing changes to component: %v of type %s", *(po.localConfig.ComponentSettings.ComponentName), po.sourceType)

	switch po.sourceType {
	case string(util.LOCAL), string(util.BINARY):
		// use value of '--dir' as source if it was used

		if po.sourceType == string(util.LOCAL) {
			glog.V(4).Infof("Copying directory %s to pod", po.sourcePath)
			err = component.PushLocal(
				po.client,
				*(po.localConfig.ComponentSettings.ComponentName),
				*(po.localConfig.ComponentSettings.App),
				po.sourcePath,
				os.Stdout,
				[]string{},
				[]string{},
				true,
				util.GetAbsGlobExps(po.sourcePath, po.ignores),
			)
		} else {
			dir := filepath.Dir(po.sourcePath)
			glog.V(4).Infof("Copying file %s to pod", po.sourcePath)
			err = component.PushLocal(
				po.client,
				*(po.localConfig.ComponentSettings.ComponentName),
				*(po.localConfig.ComponentSettings.App),
				dir,
				os.Stdout,
				[]string{po.sourcePath},
				[]string{},
				true,
				util.GetAbsGlobExps(po.sourcePath, po.ignores),
			)
		}
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Failed to push component: %v", *(po.localConfig.ComponentSettings.ComponentName)))
		}

	case "git":
		// currently we don't support changing build type
		// it doesn't make sense to use --dir with git build
		if len(po.local) != 0 {
			log.Errorf("Unable to push local directory:%s to component %s that uses Git repository:%s.", po.local, *(po.localConfig.ComponentSettings.ComponentName), po.sourcePath)
			os.Exit(1)
		}
		err := component.Build(
			po.client,
			*(po.localConfig.ComponentSettings.ComponentName),
			*(po.localConfig.ComponentSettings.App),
			true,
			stdout,
		)
		return errors.Wrapf(err, fmt.Sprintf("failed to push component: %v", *(po.localConfig.ComponentSettings.ComponentName)))
	}

	log.Successf("Changes successfully pushed to component: %v", *(po.localConfig.ComponentSettings.ComponentName))

	return
}

// NewCmdPush implements the push odo command
func NewCmdPush(name, fullName string) *cobra.Command {
	po := NewPushOptions()

	var pushCmd = &cobra.Command{
		Use:     fmt.Sprintf("%s [component name]", name),
		Short:   "Push source code to a component",
		Long:    `Push source code to a component.`,
		Example: fmt.Sprintf(pushCmdExample, fullName),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(po, cmd, args)
		},
	}

	pushCmd.Flags().StringVarP(&po.componentContext, "context", "c", "", "Use given context directory as a source for component settings")
	pushCmd.Flags().StringSliceVar(&po.ignores, "ignore", []string{}, "Files or folders to be ignored via glob expressions.")

	// Add a defined annotation in order to appear in the help menu
	pushCmd.Annotations = map[string]string{"command": "component"}
	pushCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)
	completion.RegisterCommandHandler(pushCmd, completion.ComponentNameCompletionHandler)

	return pushCmd
}
