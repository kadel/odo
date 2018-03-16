package occlient

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	appsclientset "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	buildschema "github.com/openshift/client-go/build/clientset/versioned/scheme"
	buildclientset "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imageclientset "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"

	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openshift/source-to-image/pkg/tar"
	s2ifs "github.com/openshift/source-to-image/pkg/util/fs"
)

const ocRequestTimeout = 1 * time.Second

// ocpath stores the path to oc binary
var ocpath string

type OpenShiftClient struct {
}

// parseImageName parse image reference
// returns (imageName, tag, digest, error)
// if image is referenced by tag (name:tag)  than digest is ""
// if image is referenced by digest (name@digest) than  tag is ""
func parseImageName(image string) (string, string, string, error) {
	digestParts := strings.Split(image, "@")
	if len(digestParts) == 2 {
		// image is references digest
		return digestParts[0], "", digestParts[1], nil
	} else if len(digestParts) == 1 {
		tagParts := strings.Split(image, ":")
		if len(tagParts) == 2 {
			// image references tag
			return tagParts[0], tagParts[1], "", nil
		} else if len(tagParts) == 1 {
			return tagParts[0], "latest", "", nil
		}
	}
	return "", "", "", fmt.Errorf("invalid image reference %s", image)

}

func initialize() error {
	// don't execute further if ocpath was already set
	if ocpath != "" {
		return nil
	}

	var err error
	ocpath, err = getOcBinary()
	if err != nil {
		return errors.Wrap(err, "unable to get oc binary")
	}
	if !isServerUp() {
		return errors.New("server is down")
	}
	if !isLoggedIn() {
		return errors.New("please log in to the cluster")
	}
	return nil
}

// getOcBinary returns full path to oc binary
// first it looks for env variable KUBECTL_PLUGINS_CALLER (run as oc plugin)
// than looks for env variable OC_BIN (set manualy by user)
// at last it tries to find oc in default PATH
func getOcBinary() (string, error) {
	log.Debug("getOcBinary - searching for oc binary")

	var ocPath string

	envKubectlPluginCaller := os.Getenv("KUBECTL_PLUGINS_CALLER")
	envOcBin := os.Getenv("OC_BIN")

	log.Debugf("envKubectlPluginCaller = %s\n", envKubectlPluginCaller)
	log.Debugf("envOcBin = %s\n", envOcBin)

	if len(envKubectlPluginCaller) > 0 {
		log.Debug("using path from KUBECTL_PLUGINS_CALLER")
		ocPath = envKubectlPluginCaller
	} else if len(envOcBin) > 0 {
		log.Debug("using path from OC_BIN")
		ocPath = envOcBin
	} else {
		path, err := exec.LookPath("oc")
		if err != nil {
			log.Debug("oc binary not found in PATH")
			return "", err
		}
		log.Debug("using oc from PATH")
		ocPath = path
	}
	log.Debug("using oc from %s", ocPath)

	if _, err := os.Stat(ocPath); err != nil {
		return "", err
	}

	return ocPath, nil
}

type OcCommand struct {
	args   []string
	data   *string
	format string
}

// runOcCommands executes oc
// args - command line arguments to be passed to oc ('-o json' is added by default if data is not nil)
// data - is a pointer to a string, if set than data is given to command to stdin ('-f -' is added to args as default)
func runOcComamnd(command *OcCommand) ([]byte, error) {
	if err := initialize(); err != nil {
		return nil, errors.Wrap(err, "unable to perform oc initializations")
	}

	cmd := exec.Command(ocpath, command.args...)

	// if data is not set assume that it is get command
	if len(command.format) > 0 {
		cmd.Args = append(cmd.Args, "-o", command.format)
	}
	if command.data != nil {
		// data is given, assume this is crate or apply command
		// that takes data from stdin
		cmd.Args = append(cmd.Args, "-f", "-")

		// Read from stdin
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}

		// Write to stdin
		go func() {
			defer stdin.Close()
			_, err := io.WriteString(stdin, *command.data)
			if err != nil {
				fmt.Printf("can't write to stdin %v\n", err)
			}
		}()
	}

	log.Debugf("running oc command with arguments: %s\n", strings.Join(cmd.Args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil, errors.Wrapf(err, "command: %v failed to run:\n%v", cmd.Args, string(output))
		}
		return nil, errors.Wrap(err, "unable to get combined output")
	}

	return output, nil
}

func isLoggedIn() bool {
	cmd := exec.Command(ocpath, "whoami")
	output, err := cmd.CombinedOutput()
	log.Debugf("isLoggedIn err:  %#v \n output: %#v", err, string(output))
	if err != nil {
		log.Debug(errors.Wrap(err, "error running command"))
		log.Debugf("Output is: %v", output)
		return false
	}
	return true
}

func isServerUp() bool {
	cmd := exec.Command(ocpath, "whoami", "--show-server")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug(errors.Wrap(err, "error running command"))
		return false
	}

	server := strings.TrimSpace(string(output))
	u, err := url.Parse(server)
	if err != nil {
		log.Debug(errors.Wrap(err, "unable to parse url"))
		return false
	}

	log.Debugf("Trying to connect to server %v - %v", u.Host)
	_, connectionError := net.DialTimeout("tcp", u.Host, time.Duration(ocRequestTimeout))
	if connectionError != nil {
		log.Debug(errors.Wrap(connectionError, "unable to connect to server"))
		return false
	}
	log.Debugf("Server %v is up", server)
	return true
}

func GetCurrentProjectName() (string, error) {
	// We need to run `oc project` because it returns an error when project does
	// not exist, while `oc project -q` does not return an error, it simply
	// returns the project name
	_, err := runOcComamnd(&OcCommand{
		args: []string{"project"},
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to get current project info")
	}

	output, err := runOcComamnd(&OcCommand{
		args: []string{"project", "-q"},
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to get current project name")
	}

	return strings.TrimSpace(string(output)), nil
}

func GetProjects() (string, error) {
	output, err := runOcComamnd(&OcCommand{
		args:   []string{"get", "project"},
		format: "custom-columns=NAME:.metadata.name",
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to get projects")
	}
	return strings.Join(strings.Split(string(output), "\n")[1:], "\n"), nil
}

func CreateNewProject(name string) error {
	_, err := runOcComamnd(&OcCommand{
		args: []string{"new-project", name},
	})
	if err != nil {
		return errors.Wrap(err, "unable to create new project")
	}
	return nil
}

// addLabelsToArgs adds labels from map to args as a new argument in format that oc requires
// --labels label1=value1,label2=value2
func addLabelsToArgs(labels map[string]string, args []string) []string {
	if labels != nil {
		var labelsString []string
		for key, value := range labels {
			labelsString = append(labelsString, fmt.Sprintf("%s=%s", key, value))
		}
		args = append(args, "--labels")
		args = append(args, strings.Join(labelsString, ","))
	}

	return args
}

// NewAppS2I create new application  using S2I with source in git repository
func NewAppS2I(name string, builderImage string, gitUrl string, labels map[string]string) (string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		panic(err.Error())
	}
	namespace, _, _ := kubeConfig.Namespace()

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	imageClient, err := imageclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	appsClient, err := appsclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	buildClient, err := buildclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	imageName, imageTag, _, err := parseImageName(builderImage)
	if err != nil {
		return "", errors.Wrap(err, "unable to create new s2i build")
	}
	log.Debugf("Checking for exact match with ImageStream")
	var exactMatchName bool
	imageStream, err := imageClient.ImageStreams("openshift").Get(imageName, metav1.GetOptions{})
	if err != nil {
		log.Debugf("No exact match found: %s", err.Error())
		exactMatchName = false
	} else {
		exactMatchName = true
	}
	if exactMatchName {
		for _, tag := range imageStream.Status.Tags {
			if tag.Tag == imageTag {
				log.Debugf("Found exact image tag match for %s", imageTag)
				// first item is the latest one
				tagDigest := tag.Items[0].Image
				imageStreamImage, err := imageClient.ImageStreamImages("openshift").Get(fmt.Sprintf("%s@%s", imageName, tagDigest), metav1.GetOptions{})
				if err != nil {
					panic(err)
				}
				//TODO determine what port should be exposed
				fmt.Printf("%#v\n", imageStreamImage.Image)
			}
		}
	}

	is := imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err = imageClient.ImageStreams(namespace).Create(&is)
	if err != nil {
		panic(err.Error())
	}

	bc := buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						// TODO
						Name: name + ":latest",
					},
				},
				Source: buildv1.BuildSource{
					Git: &buildv1.GitBuildSource{
						URI: gitUrl,
					},
					Type: "Git",
				},
				Strategy: buildv1.BuildStrategy{
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							// TODO
							Name: "nodejs:latest",
							// TODO
							Namespace: "openshift",
						},
					},
				},
			},
			Triggers: []buildv1.BuildTriggerPolicy{
				buildv1.BuildTriggerPolicy{
					Type: "ConfigChange",
				},
				buildv1.BuildTriggerPolicy{
					Type:        "ImageChange",
					ImageChange: &buildv1.ImageChangeTrigger{},
				},
			},
		},
	}
	_, err = buildClient.BuildConfigs(namespace).Create(&bc)
	if err != nil {
		panic(err.Error())
	}

	dc := appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"deploymentconfig": name,
			},
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: map[string]string{
				"deploymentconfig": name,
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deploymentconfig": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Image: name + ":latest",
							Name:  name,
						},
					},
				},
			},
			Triggers: []appsv1.DeploymentTriggerPolicy{
				appsv1.DeploymentTriggerPolicy{
					Type: "ConfigChange",
				},
				appsv1.DeploymentTriggerPolicy{
					Type: "ImageChange",
					ImageChangeParams: &appsv1.DeploymentTriggerImageChangeParams{
						Automatic: true,
						ContainerNames: []string{
							name,
						},
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: name + ":latest",
						},
					},
				},
			},
		},
	}
	_, err = appsClient.DeploymentConfigs(namespace).Create(&dc)
	if err != nil {
		panic(err.Error())
	}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					// TODO
					Name:       "8080-tcp",
					Port:       8080,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: map[string]string{
				"deploymentconfig": name,
			},
		},
	}

	_, err = kubeClient.CoreV1().Services(namespace).Create(&svc)
	if err != nil {
		panic(err.Error())
	}

	return "", nil

}

// NewAppS2I create new application  using S2I from local directory
func NewAppS2IEmpty(name string, builderImage string, labels map[string]string) (string, error) {

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		panic(err.Error())
	}
	namespace, _, _ := kubeConfig.Namespace()

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	imageClient, err := imageclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	appsClient, err := appsclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	buildClient, err := buildclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	imageName, imageTag, _, err := parseImageName(builderImage)
	if err != nil {
		return "", errors.Wrap(err, "unable to create new s2i build")
	}
	log.Debugf("Checking for exact match with ImageStream")
	var exactMatchName bool
	imageStream, err := imageClient.ImageStreams("openshift").Get(imageName, metav1.GetOptions{})
	if err != nil {
		log.Debugf("No exact match found: %s", err.Error())
		exactMatchName = false
	} else {
		exactMatchName = true
	}
	if exactMatchName {
		for _, tag := range imageStream.Status.Tags {
			if tag.Tag == imageTag {
				log.Debugf("Found exact image tag match for %s", imageTag)
				// first item is the latest one
				tagDigest := tag.Items[0].Image
				imageStreamImages, err := imageClient.ImageStreamImages("openshift").Get(fmt.Sprintf("%s@%s", imageName, tagDigest), metav1.GetOptions{})
				if err != nil {
					panic(err)
				}
				fmt.Printf("%#v\n", imageStreamImages)
			}
		}
	}

	is := imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err = imageClient.ImageStreams(namespace).Create(&is)
	if err != nil {
		panic(err.Error())
	}

	bc := buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						// TODO
						Name: name + ":latest",
					},
				},
				Source: buildv1.BuildSource{
					Type:   "Binary",
					Binary: &buildv1.BinaryBuildSource{},
				},
				Strategy: buildv1.BuildStrategy{
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							// TODO
							Name: "nodejs:latest",
							// TODO
							Namespace: "openshift",
						},
					},
				},
			},
			Triggers: []buildv1.BuildTriggerPolicy{
				buildv1.BuildTriggerPolicy{
					Type: "ConfigChange",
				},
				buildv1.BuildTriggerPolicy{
					Type:        "ImageChange",
					ImageChange: &buildv1.ImageChangeTrigger{},
				},
			},
		},
	}
	_, err = buildClient.BuildConfigs(namespace).Create(&bc)
	if err != nil {
		panic(err.Error())
	}

	dc := appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"deploymentconfig": name,
			},
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: map[string]string{
				"deploymentconfig": name,
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deploymentconfig": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Image: name + ":latest",
							Name:  name,
						},
					},
				},
			},
			Triggers: []appsv1.DeploymentTriggerPolicy{
				appsv1.DeploymentTriggerPolicy{
					Type: "ConfigChange",
				},
				appsv1.DeploymentTriggerPolicy{
					Type: "ImageChange",
					ImageChangeParams: &appsv1.DeploymentTriggerImageChangeParams{
						Automatic: true,
						ContainerNames: []string{
							name,
						},
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: name + ":latest",
						},
					},
				},
			},
		},
	}
	_, err = appsClient.DeploymentConfigs(namespace).Create(&dc)
	if err != nil {
		panic(err.Error())
	}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					// TODO
					Name:       "8080-tcp",
					Port:       8080,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: map[string]string{
				"deploymentconfig": name,
			},
		},
	}

	_, err = kubeClient.CoreV1().Services(namespace).Create(&svc)
	if err != nil {
		panic(err.Error())
	}

	return "", nil

}

func StartBuild(name string, dir string) (string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		panic(err.Error())
	}
	namespace, _, _ := kubeConfig.Namespace()

	buildClient, err := buildclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	var r io.Reader
	pr, pw := io.Pipe()
	go func() {
		w := gzip.NewWriter(pw)
		if err := tar.New(s2ifs.NewFileSystem()).CreateTarStream(dir, false, w); err != nil {
			pw.CloseWithError(err)
		} else {
			w.Close()
			pw.CloseWithError(io.EOF)
		}
	}()
	r = pr

	buildRequest := buildv1.BuildRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	result := &buildv1.Build{}
	// this should be  buildClient.BuildConfigs(namespace).Instantiate
	// but there is no way to pass data using that call.
	err = buildClient.RESTClient().Post().
		Namespace(namespace).
		Resource("buildconfigs").
		Name(name).
		SubResource("instantiatebinary").
		Body(r).
		VersionedParams(&buildRequest, buildschema.ParameterCodec).
		Do().
		Into(result)

	return "", nil

}

// Delete calls oc delete
// kind is always required (can be set to 'all')
// name can be omitted if labels are set, in that case set name to ''
// if you want to delete object just by its name set labels to nil
func Delete(kind string, name string, labels map[string]string) (string, error) {

	args := []string{
		"delete",
		kind,
	}

	if len(name) > 0 {
		args = append(args, name)
	}

	if labels != nil {
		var labelsString []string
		for key, value := range labels {
			labelsString = append(labelsString, fmt.Sprintf("%s=%s", key, value))
		}
		args = append(args, "--selector")
		args = append(args, strings.Join(labelsString, ","))
	}

	output, err := runOcComamnd(&OcCommand{args: args})
	if err != nil {
		return "", err
	}

	return string(output[:]), nil

}

func DeleteProject(name string) error {
	_, err := runOcComamnd(&OcCommand{
		args: []string{"delete", "project", name},
	})
	if err != nil {
		return errors.Wrap(err, "unable to delete project")
	}
	return nil
}

type VolumeConfig struct {
	Name             *string
	Size             *string
	DeploymentConfig *string
	Path             *string
}

type VolumeOpertaions struct {
	Add    bool
	Remove bool
	List   bool
}

func SetVolumes(config *VolumeConfig, operations *VolumeOpertaions) (string, error) {
	args := []string{
		"set",
		"volumes",
		"dc", *config.DeploymentConfig,
		"--type", "pvc",
	}
	if config.Name != nil {
		args = append(args, "--name", *config.Name)
	}
	if config.Size != nil {
		args = append(args, "--claim-size", *config.Size)
	}
	if config.Path != nil {
		args = append(args, "--mount-path", *config.Path)
	}
	if operations.Add {
		args = append(args, "--add")
	}
	if operations.Remove {
		args = append(args, "--remove", "--confirm")
	}
	if operations.List {
		args = append(args, "--all")
	}
	output, err := runOcComamnd(&OcCommand{
		args: args,
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to perform volume operations")
	}
	return string(output), nil
}

// GetLabelValues get label values from all object that are labeled with given label
// returns slice of uniq label values
func GetLabelValues(project string, label string) ([]string, error) {
	// get all object that have given label
	// and show just label values separated by ,
	args := []string{
		"get", "all",
		"--selector", label,
		"--namespace", project,
		"-o", "go-template={{range .items}}{{range $key, $value := .metadata.labels}}{{if eq $key \"" + label + "\"}}{{$value}},{{end}}{{end}}{{end}}",
	}

	output, err := runOcComamnd(&OcCommand{args: args})
	if err != nil {
		return nil, err
	}

	values := []string{}
	// deduplicate label values
	for _, val := range strings.Split(string(output), ",") {
		val = strings.TrimSpace(val)
		if val != "" {
			// check if this is the first time we see this value
			found := false
			for _, existing := range values {
				if val == existing {
					found = true
				}
			}
			if !found {
				values = append(values, val)
			}
		}
	}

	return values, nil
}
