package component

import (
	"fmt"
	"os"
	"strings"

	"github.com/redhat-developer/odo/pkg/config"
	"github.com/redhat-developer/odo/pkg/odo/cli/component/ui"
	commonui "github.com/redhat-developer/odo/pkg/odo/cli/ui"

	"github.com/pkg/errors"
	appCmd "github.com/redhat-developer/odo/pkg/odo/cli/application"
	projectCmd "github.com/redhat-developer/odo/pkg/odo/cli/project"

	"github.com/redhat-developer/odo/pkg/log"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	odoutil "github.com/redhat-developer/odo/pkg/odo/util"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"
	"github.com/redhat-developer/odo/pkg/odo/util/validation"

	"github.com/redhat-developer/odo/pkg/catalog"
	"github.com/redhat-developer/odo/pkg/component"
	catalogutil "github.com/redhat-developer/odo/pkg/odo/cli/catalog/util"
	"github.com/redhat-developer/odo/pkg/util"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

// CreateOptions encapsulates create options
type CreateOptions struct {
	localConfig *config.LocalConfigInfo
	*genericclioptions.Context
	componentBinary  string
	componentGit     string
	componentGitRef  string
	componentContext string
	componentPorts   []string
	componentEnvVars []string
	memoryMax        string
	memoryMin        string
	memory           string
	cpuMax           string
	cpuMin           string
	cpu              string
	wait             bool
	interactive      bool
}

// CreateRecommendedCommandName is the recommended watch command name
const CreateRecommendedCommandName = "create"

var createLongDesc = ktemplates.LongDesc(`Create a configuration describing component to be deployed by  on OpenShift.

If a component name is not provided, it'll be auto-generated.

By default, builder images will be used from the current namespace. You can explicitly supply a namespace by using: odo create namespace/name:version
If version is not specified by default, latest wil be chosen as the version.

A full list of component types that can be deployed is available using: 'odo catalog list'`)

var createExample = ktemplates.Examples(`  # Create new Node.js component with the source in current directory.
%[1]s nodejs

# A specific image version may also be specified
%[1]s nodejs:latest

# Create new Node.js component named 'frontend' with the source in './frontend' directory
%[1]s nodejs frontend --local ./frontend

# Create a new Node.js component of version 6 from the 'openshift' namespace
%[1]s openshift/nodejs:6 --local /nodejs-ex

# Create new Wildfly component with binary named sample.war in './downloads' directory
%[1]s wildfly wildly --binary ./downloads/sample.war

# Create new Node.js component with source from remote git repository
%[1]s nodejs --git https://github.com/openshift/nodejs-ex.git

# Create new Node.js git component while specifying a branch, tag or commit ref
%[1]s nodejs --git https://github.com/openshift/nodejs-ex.git --ref master

# Create new Node.js git component while specifying a tag
%[1]s nodejs --git https://github.com/openshift/nodejs-ex.git --ref v1.0.1

# Create new Node.js component with the source in current directory and ports 8080-tcp,8100-tcp and 9100-udp exposed
%[1]s nodejs --port 8080,8100/tcp,9100/udp

# Create new Node.js component with the source in current directory and env variables key=value and key1=value1 exposed
%[1]s nodejs --env key=value,key1=value1

# For more examples, visit: https://github.com/redhat-developer/odo/blob/master/docs/examples.md
%[1]s python --git https://github.com/openshift/django-ex.git

# Passing memory limits
%[1]s nodejs --memory 150Mi
%[1]s nodejs --min-memory 150Mi --max-memory 300 Mi

# Passing cpu limits
%[1]s nodejs --cpu 2
%[1]s nodejs --min-cpu 200m --max-cpu 2

  `)

// NewCreateOptions returns new instance of CreateOptions
func NewCreateOptions() *CreateOptions {
	return &CreateOptions{}
}

func (co *CreateOptions) setCmpSourceAttrs() (err error) {

	if len(co.componentContext) != 0 {
		err = co.localConfig.GetLocalConfigFileFromPath(co.componentContext)
		if err != nil {
			return errors.Wrap(err, "failed intialising component config file")
		}
	}

	componentCnt := 0
	localSrcCmp := string(util.LOCAL)
	co.localConfig.ComponentSettings.Type = &localSrcCmp

	if len(co.componentBinary) != 0 {
		cPath, err := util.GetAbsPath(co.componentBinary)
		if err != nil {
			return err
		}
		co.localConfig.ComponentSettings.Path = &cPath
		binarySrcCmp := string(util.BINARY)
		co.localConfig.ComponentSettings.Type = &binarySrcCmp
		componentCnt++
	}
	if len(co.componentGit) != 0 {
		co.localConfig.ComponentSettings.Path = &(co.componentGit)
		gitSrcCmp := string(util.GIT)
		co.localConfig.ComponentSettings.Type = &gitSrcCmp
		componentCnt++
	}

	if componentCnt > 1 {
		return fmt.Errorf("The source can be either --binary or --local or --git")
	}

	if len(co.componentGitRef) != 0 {
		co.localConfig.ComponentSettings.Ref = &(co.componentGitRef)
	}

	if len(co.componentGit) == 0 && len(co.componentGitRef) != 0 {
		return fmt.Errorf("The --ref flag is only valid for --git flag")
	}

	return
}

func (co *CreateOptions) setCmpName(args []string) (err error) {
	componentImageName, componentType, _, _ := util.ParseComponentImageName(args[0])
	co.localConfig.ComponentSettings.ComponentType = &componentImageName

	if len(args) == 2 {
		co.localConfig.ComponentSettings.ComponentName = &args[1]
		return
	}

	cmpSrcType, err := util.GetCreateType(*(co.localConfig.ComponentSettings.Type))
	if err != nil {
		return errors.Wrap(err, "failed to generate a name for component")
	}
	componentName, err := createDefaultComponentName(
		co.Context,
		componentType,
		cmpSrcType,
		co.componentContext,
	)
	if err != nil {
		return err
	}

	co.localConfig.ComponentSettings.ComponentName = &componentName
	return
}

func createDefaultComponentName(context *genericclioptions.Context, componentType string, sourceType util.CreateType, sourcePath string) (string, error) {
	// Fetch list of existing components in-order to attempt generation of unique component name
	componentList, err := component.List(context.Client, context.Application)
	if err != nil {
		return "", err
	}

	// Retrieve the componentName, if the componentName isn't specified, we will use the default image name
	componentName, err := component.GetDefaultComponentName(
		sourcePath,
		sourceType,
		componentType,
		componentList,
	)

	if err != nil {
		return "", nil
	}

	return componentName, nil
}

func (co *CreateOptions) setResourceLimits() {
	ensureAndLogProperResourceUsage(co.memory, co.memoryMin, co.memoryMax, "memory")

	ensureAndLogProperResourceUsage(co.cpu, co.cpuMin, co.cpuMax, "cpu")

	memoryQuantity := util.FetchResourceQuantity(corev1.ResourceMemory, co.memoryMin, co.memoryMax, co.memory)
	if memoryQuantity != nil {
		minMemory := memoryQuantity.MinQty.String()
		maxMemory := memoryQuantity.MaxQty.String()
		co.localConfig.ComponentSettings.MinMemory = &minMemory
		co.localConfig.ComponentSettings.MaxMemory = &maxMemory
	}

	cpuQuantity := util.FetchResourceQuantity(corev1.ResourceCPU, co.cpuMin, co.cpuMax, co.cpu)
	if cpuQuantity != nil {
		minCPU := cpuQuantity.MinQty.String()
		maxCPU := cpuQuantity.MaxQty.String()
		co.localConfig.ComponentSettings.MinCPU = &minCPU
		co.localConfig.ComponentSettings.MaxCPU = &maxCPU
	}
}

// Complete completes create args
func (co *CreateOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	if len(args) == 0 || !cmd.HasFlags() {
		co.interactive = true
	}

	co.localConfig, err = config.NewLocalConfigInfo()
	if err != nil {
		return errors.Wrap(err, "failed intiating local config")
	}

	co.Context = genericclioptions.NewContextCreatingAppIfNeeded(cmd)
	co.localConfig.ComponentSettings.App = &(co.Context.Application)

	if co.interactive {
		client := co.Client

		componentTypeCandidates, err := catalog.List(client)
		if err != nil {
			return err
		}
		componentTypeCandidates = catalogutil.FilterHiddenComponents(componentTypeCandidates)
		selectedComponentType := ui.SelectComponentType(componentTypeCandidates)
		selectedImageTag := ui.SelectImageTag(componentTypeCandidates, selectedComponentType)
		componentType := selectedComponentType + ":" + selectedImageTag
		co.localConfig.ComponentSettings.ComponentType = &componentType

		selectedSourceType := ui.SelectSourceType([]util.CreateType{util.LOCAL, util.GIT, util.BINARY})
		componentSrcType := string(selectedSourceType)
		co.localConfig.ComponentSettings.Type = &componentSrcType
		selectedSourcePath := ""
		currentDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		if selectedSourceType == util.BINARY {
			selectedSourcePath = ui.EnterInputTypePath("binary", currentDirectory)
			selectedSourcePath, err = util.GetAbsPath(selectedSourcePath)
			if err != nil {
				return err
			}
		} else if selectedSourceType == util.GIT {
			var selectedGitRef string
			selectedSourcePath, selectedGitRef = ui.EnterGitInfo()
			co.localConfig.ComponentSettings.Ref = &selectedGitRef
		}
		co.localConfig.ComponentSettings.Path = &selectedSourcePath

		defaultComponentName, err := createDefaultComponentName(co.Context, selectedComponentType, selectedSourceType, selectedSourcePath)
		if err != nil {
			return err
		}
		componentName := ui.EnterComponentName(defaultComponentName, co.Context)
		co.localConfig.ComponentSettings.ComponentName = &componentName

		if commonui.Proceed("Do you wish to set advanced options") {
			ports := ui.EnterPorts()
			if len(ports) > 0 {
				portsStr := strings.Join(ports, " ")
				co.localConfig.ComponentSettings.Ports = &portsStr
			}
			co.componentEnvVars = ui.EnterEnvVars()

			if commonui.Proceed("Do you wish to set resource limits") {
				memMax := ui.EnterMemory("maximum", "512Mi")
				memMin := ui.EnterMemory("minimum", memMax)
				cpuMax := ui.EnterCPU("maximum", "1")
				cpuMin := ui.EnterCPU("minimum", cpuMax)

				memoryQuantity := util.FetchResourceQuantity(corev1.ResourceMemory, memMin, memMax, "")
				if memoryQuantity != nil {
					co.localConfig.ComponentSettings.MinMemory = &memMin
					co.localConfig.ComponentSettings.MaxMemory = &memMax
				}
				cpuQuantity := util.FetchResourceQuantity(corev1.ResourceCPU, cpuMin, cpuMax, "")
				if cpuQuantity != nil {
					co.localConfig.ComponentSettings.MinCPU = &cpuMin
					co.localConfig.ComponentSettings.MaxCPU = &cpuMax
				}
			}
		}
	} else {
		err = co.setCmpSourceAttrs()
		if err != nil {
			return err
		}
		err = co.setCmpName(args)
		if err != nil {
			return err
		}
		co.setResourceLimits()
		if len(co.componentPorts) > 0 {
			portsStr := strings.Join(co.componentPorts, " ")
			co.localConfig.ComponentSettings.Ports = &portsStr
		}
	}

	co.localConfig.ComponentSettings.Project = &(co.Context.Project)

	return
}

// Validate validates the create parameters
func (co *CreateOptions) Validate() (err error) {
	_, componentType, _, componentVersion := util.ParseComponentImageName(*(co.localConfig.ComponentSettings.ComponentType))
	// Check to see if the catalog type actually exists
	exists, err := catalog.Exists(co.Context.Client, componentType)
	if err != nil {
		return errors.Wrapf(err, "Failed to create component of type %s", componentType)
	}
	if !exists {
		log.Info("Run 'odo catalog list components' for a list of supported component types")
		return fmt.Errorf("Failed to find component of type %s", componentType)
	}

	// Check to see if that particular version exists
	versionExists, err := catalog.VersionExists(co.Context.Client, componentType, componentVersion)
	if err != nil {
		return errors.Wrapf(err, "Failed to create component of type %s of version %s", componentType, componentVersion)
	}
	if !versionExists {
		log.Info("Run 'odo catalog list components' to see a list of supported component type versions")
		return fmt.Errorf("Invalid component version %s:%s", componentType, componentVersion)
	}

	// Validate component name
	err = validation.ValidateName(*(co.localConfig.ComponentSettings.ComponentName))
	if err != nil {
		return errors.Wrapf(err, "failed to create component of name %s", *(co.localConfig.ComponentSettings.ComponentName))
	}

	exists, err = component.Exists(co.Context.Client, *(co.localConfig.ComponentSettings.ComponentName), co.Context.Application)
	if err != nil {
		return errors.Wrapf(err, "failed to check if component of name %s exists in application %s", *(co.localConfig.ComponentSettings.ComponentName), co.Context.Application)
	}
	if exists {
		return fmt.Errorf("component with name %s already exists in application %s", *(co.localConfig.ComponentSettings.ComponentName), co.Context.Application)
	}

	*(co.localConfig.ComponentSettings.App) = co.Context.Application
	return
}

// Run has the logic to perform the required actions as part of command
func (co *CreateOptions) Run() (err error) {
	if co.localConfig.ComponentSettings.ComponentType != nil {
		err = co.localConfig.SetConfiguration("ComponentType", *(co.localConfig.ComponentSettings.ComponentType))
	}
	if co.localConfig.ComponentSettings.App != nil {
		err = co.localConfig.SetConfiguration("App", *(co.localConfig.ComponentSettings.App))
	}
	if co.localConfig.ComponentSettings.ComponentName != nil {
		err = co.localConfig.SetConfiguration("ComponentName", *(co.localConfig.ComponentSettings.ComponentName))
	}
	if co.localConfig.ComponentSettings.MaxCPU != nil {
		err = co.localConfig.SetConfiguration("MaxCPU", *(co.localConfig.ComponentSettings.MaxCPU))
	}
	if co.localConfig.ComponentSettings.MaxMemory != nil {
		err = co.localConfig.SetConfiguration("MaxMemory", *(co.localConfig.ComponentSettings.MaxMemory))
	}
	if co.localConfig.ComponentSettings.MinCPU != nil {
		err = co.localConfig.SetConfiguration("MinCPU", *(co.localConfig.ComponentSettings.MinCPU))
	}
	if co.localConfig.ComponentSettings.MinMemory != nil {
		err = co.localConfig.SetConfiguration("MinMemory", *(co.localConfig.ComponentSettings.MinMemory))
	}
	if co.localConfig.ComponentSettings.Path != nil {
		err = co.localConfig.SetConfiguration("Path", *(co.localConfig.ComponentSettings.Path))
	}
	if co.localConfig.ComponentSettings.Ports != nil {
		err = co.localConfig.SetConfiguration("Ports", *(co.localConfig.ComponentSettings.Ports))
	}
	if co.localConfig.ComponentSettings.Ref != nil {
		err = co.localConfig.SetConfiguration("Ref", *(co.localConfig.ComponentSettings.Ref))
	}
	if co.localConfig.ComponentSettings.Type != nil {
		err = co.localConfig.SetConfiguration("Type", *(co.localConfig.ComponentSettings.Type))
	}
	if co.localConfig.ComponentSettings.Project != nil {
		err = co.localConfig.SetConfiguration("Project", *(co.localConfig.ComponentSettings.Project))
	}
	return
}

// The general cpu/memory is used as a fallback when it's set and both min-cpu/memory max-cpu/memory are not set
// when the only thing specified is the min or max value, we exit the application
func ensureAndLogProperResourceUsage(resource, resourceMin, resourceMax, resourceName string) {
	if resourceMin != "" && resourceMax != "" && resource != "" {
		log.Infof("`--%s` will be ignored as `--min-%s` and `--max-%s` has been passed\n", resourceName, resourceName, resourceName)
	}
	if (resourceMin == "") != (resourceMax == "") && resource != "" {
		log.Infof("Using `--%s` %s for min and max limits.\n", resourceName, resource)
	}
	if (resourceMin == "") != (resourceMax == "") && resource == "" {
		log.Errorf("`--min-%s` should accompany `--max-%s` or pass `--%s` to use same value for both min and max or try not passing any of them\n", resourceName, resourceName, resourceName)
		os.Exit(1)
	}
}

// NewCmdCreate implements the create odo command
func NewCmdCreate(name, fullName string) *cobra.Command {
	co := NewCreateOptions()
	var componentCreateCmd = &cobra.Command{
		Use:     fmt.Sprintf("%s <component_type> [component_name] [flags]", name),
		Short:   "Create a new component",
		Long:    createLongDesc,
		Example: fmt.Sprintf(createExample, fullName),
		Args:    cobra.RangeArgs(0, 2),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(co, cmd, args)
		},
	}
	componentCreateCmd.Flags().StringVarP(&co.componentBinary, "binary", "b", "", "Use a binary as the source file for the component")
	componentCreateCmd.Flags().StringVarP(&co.componentGit, "git", "g", "", "Use a git repository as the source file for the component")
	componentCreateCmd.Flags().StringVarP(&co.componentGitRef, "ref", "r", "", "Use a specific ref e.g. commit, branch or tag of the git repository")
	componentCreateCmd.Flags().StringVar(&co.componentContext, "context", "", "Use local directory as a source file for the component")
	componentCreateCmd.Flags().StringVar(&co.memory, "memory", "", "Amount of memory to be allocated to the component. ex. 100Mi")
	componentCreateCmd.Flags().StringVar(&co.memoryMin, "min-memory", "", "Limit minimum amount of memory to be allocated to the component. ex. 100Mi")
	componentCreateCmd.Flags().StringVar(&co.memoryMax, "max-memory", "", "Limit maximum amount of memory to be allocated to the component. ex. 100Mi")
	componentCreateCmd.Flags().StringVar(&co.cpu, "cpu", "", "Amount of cpu to be allocated to the component. ex. 100m or 0.1")
	componentCreateCmd.Flags().StringVar(&co.cpuMin, "min-cpu", "", "Limit minimum amount of cpu to be allocated to the component. ex. 100m")
	componentCreateCmd.Flags().StringVar(&co.cpuMax, "max-cpu", "", "Limit maximum amount of cpu to be allocated to the component. ex. 1")
	componentCreateCmd.Flags().StringSliceVarP(&co.componentPorts, "port", "p", []string{}, "Ports to be used when the component is created (ex. 8080,8100/tcp,9100/udp)")
	componentCreateCmd.Flags().StringSliceVar(&co.componentEnvVars, "env", []string{}, "Environmental variables for the component. For example --env VariableName=Value")

	// Add a defined annotation in order to appear in the help menu
	componentCreateCmd.Annotations = map[string]string{"command": "component"}
	componentCreateCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(componentCreateCmd)
	//Adding `--application` flag
	appCmd.AddApplicationFlag(componentCreateCmd)

	completion.RegisterCommandHandler(componentCreateCmd, completion.CreateCompletionHandler)
	completion.RegisterCommandFlagHandler(componentCreateCmd, "context", completion.FileCompletionHandler)
	completion.RegisterCommandFlagHandler(componentCreateCmd, "binary", completion.FileCompletionHandler)

	return componentCreateCmd
}
