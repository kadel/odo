package devfile

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

// Load read Devfile from filename
func Load(filename string) (*Devfile, error) {

	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var data Devfile
	err = yaml.Unmarshal(f, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// EnvToEnvVar convert array oof Env from Devfile to array of k8s EnvVar
func EnvToEnvVar(envs []DockerimageEnv) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	for _, env := range envs {
		envVars = append(envVars, corev1.EnvVar{Name: *env.Name, Value: *env.Value})
	}
	return envVars
}

func (component *DevfileComponent) ConvertToContainer() (*corev1.Container, error) {
	if component.Type != DevfileComponentsTypeDockerimage {
		return nil, fmt.Errorf("component needs to have dockerimage type")
	}
	var container corev1.Container

	fmt.Printf("MountSources = %#v\n", component.MountSources)

	container.Image = *component.Image
	container.Command = component.Command
	container.Args = component.Args
	container.Env = EnvToEnvVar(component.Env)

	// TODO:
	// MemoryLimit
	// Volumes
	// Endpoints

	return &container, nil
}

// getCommandAction get information about command
// first string is command, second one is workdir
func (devf *Devfile) GetCommandAction(commandName string) (*DevfileCommandAction, error) {
	for _, command := range devf.Commands {
		if command.Name == commandName {
			if len(command.Actions) != 1 {
				return nil, fmt.Errorf("commands with only one action are supported for now")
			}
			return &command.Actions[0], nil
		}
	}
	return nil, fmt.Errorf("unable to find %s command", commandName)
}

func (devf *Devfile) GetComponent(alias string) (*DevfileComponent, error) {
	for _, component := range devf.Components {

		if component.Alias != nil && *component.Alias == alias {
			return &component, nil
		}
	}
	return nil, fmt.Errorf("unable to find component with alias=%s", alias)

}
