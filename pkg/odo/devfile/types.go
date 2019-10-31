package devfile

type Devfile struct {
	// Devfile API Version
	ApiVersion string `yaml:"apiVersion"`

	Metadata DevfileMetadata `yaml:"metadata"`

	// Description of the projects, containing names and sources locations
	Projects []DevfileProject `yaml:"projects,omitempty"`

	Attributes Attributes `yaml:"attributes,omitempty"`

	// Description of the workspace components, such as editor and plugins
	Components []DevfileComponent `yaml:"components,omitempty"`

	// Description of the predefined commands to be available in workspace
	Commands []DevfileCommand `yaml:"commands,omitempty"`
}
type DevfileMetadata struct {
	// Workspaces created from devfile, will use it as base and append random suffix.
	// It's used when name is not defined.
	GenerateName *string `yaml:"generateName,omitempty"`

	// The name of the devfile. Workspaces created from devfile, will inherit this
	// name
	Name *string `yaml:"name,omitempty"`
}

type DevfileProject struct {
	// The path relative to the root of the projects to which this project should be cloned into. This is a unix-style relative path (i.e. uses forward slashes). The path is invalid if it is absolute or tries to escape the project root through the usage of '..'. If not specified, defaults to the project name."
	ClonePath *string `yaml:"clonePath,omitempty"`

	// The Project Name
	Name string `yaml:"name"`

	// Describes the project's source - type and location
	Source DevfileProjectSource `yaml:"source"`
}

type DevfileProjectSource struct {
	Type string `yaml:"type"`

	// Project's source location address. Should be URL for git and github located projects, or file:// for zip."
	Location string `yaml:"location"`

	// The name of the of the branch to check out after obtaining the source from the location.
	//  The branch has to already exist in the source otherwise the default branch is used.
	//  In case of git, this is also the name of the remote branch to push to.
	Branch *string `yaml:"branch,omitempty"`

	// The id of the commit to reset the checked out branch to.
	//  Note that this is equivalent to 'startPoint' and provided for convenience.
	CommitId *string `yaml:"commitId,omitempty"`

	// Part of project to populate in the working directory.
	SparseCheckoutDir *string `yaml:"sparseCheckoutDir,omitempty"`

	// The tag or commit id to reset the checked out branch to.
	StartPoint *string `yaml:"startPoint,omitempty"`

	// The name of the tag to reset the checked out branch to.
	//  Note that this is equivalent to 'startPoint' and provided for convenience.
	Tag *string `yaml:"tag,omitempty"`
}

type DevfileCommand struct {
	// List of the actions of given command. Now the only one command must be
	// specified in list but there are plans to implement supporting multiple actions
	// commands.
	Actions []DevfileCommandAction `yaml:"actions"`

	// Additional command attributes
	Attributes Attributes `yaml:"attributes,omitempty"`

	// Describes the name of the command. Should be unique per commands set.
	Name string `yaml:"name"`
}

type DevfileCommandAction struct {
	// The actual action command-line string
	Command *string `yaml:"command,omitempty"`

	// Describes component to which given action relates
	Component *string `yaml:"component,omitempty"`

	// the path relative to the location of the devfile to the configuration file
	// defining one or more actions in the editor-specific format
	Reference *string `yaml:"reference,omitempty"`

	// The content of the referenced configuration file that defines one or more
	// actions in the editor-specific format
	ReferenceContent *string `yaml:"referenceContent,omitempty"`

	// Describes action type
	Type *string `yaml:"type,omitempty"`

	// Working directory where the command should be executed
	Workdir *string `yaml:"workdir,omitempty"`
}

type DevfileComponent struct {
	// The name using which other places of this devfile (like commands) can refer to
	// this component. This attribute is optional but must be unique in the devfile if
	// specified.
	Alias *string `yaml:"alias,omitempty"`

	// Describes whether projects sources should be mount to the component.
	// `CHE_PROJECTS_ROOT` environment variable should contains a path where projects
	// sources are mount
	MountSources bool `yaml:"mountSources,omitempty"`

	// Describes type of the component, e.g. whether it is an plugin or editor or
	// other type
	Type DevfileComponentsType `yaml:"type"`

	// for type=dockerfile
	DevfileComponentDockerimage `yaml:",inline"`
}

type DevfileComponentDockerimage struct {
	Image       *string               `yaml:"image,omitempty"`
	MemoryLimit *string               `yaml:"memoryLimit,omitempty"`
	Command     []string              `yaml:"command,omitempty"`
	Args        []string              `yaml:"args,omitempty"`
	Volumes     []DockerimageVolume   `yaml:"volumes,omitempty"`
	Env         []DockerimageEnv      `yaml:"env,omitempty"`
	Endpoints   []DockerimageEndpoint `yaml:"endpoints,omitempty"`
}
type DockerimageVolume struct {
	Name          *string `yaml:"name,omitempty"`
	ContainerPath *string `yaml:"containerPath,omitempty"`
}

type DockerimageEnv struct {
	Name  *string `yaml:"name,omitempty"`
	Value *string `yaml:"value,omitempty"`
}

type DockerimageEndpoint struct {
	Name *string `yaml:"name,omitempty"`
	Port *int    `yaml:"port,omitempty"`
}

type DevfileComponentsType string

const DevfileComponentsTypeCheEditor DevfileComponentsType = "cheEditor"
const DevfileComponentsTypeChePlugin DevfileComponentsType = "chePlugin"
const DevfileComponentsTypeDockerimage DevfileComponentsType = "dockerimage"
const DevfileComponentsTypeKubernetes DevfileComponentsType = "kubernetes"
const DevfileComponentsTypeOpenshift DevfileComponentsType = "openshift"

type Attributes map[string]string
