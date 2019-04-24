package auth

import (
	"os"

	oclogin "github.com/openshift/odo/pkg/auth/oclogin"
	"github.com/openshift/odo/pkg/log"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"
)

// Login takes of authentication part and returns error if there any
func Login(server, username, password, token, caAuth string, skipTLS bool) error {

	a := oclogin.LoginOptions{
		Server:         server,
		CommandName:    "odo",
		CAFile:         caAuth,
		InsecureTLS:    skipTLS,
		Username:       username,
		Password:       password,
		Project:        "",
		Token:          token,
		PathOptions:    &clientcmd.PathOptions{GlobalFile: clientcmd.RecommendedHomeFile, EnvVar: clientcmd.RecommendedConfigPathEnvVar, ExplicitFileFlag: "config", LoadingRules: &clientcmd.ClientConfigLoadingRules{ExplicitPath: ""}},
		RequestTimeout: 0,
		IOStreams:      genericclioptions.IOStreams{Out: os.Stdout, In: os.Stdin, ErrOut: os.Stderr},
	}

	// initialize client-go client and read starting kubeconfig file

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	kubeconfig, _ := kubeConfig.RawConfig()

	a.StartingKubeConfig = &kubeconfig

	// if server URL is not given as argument, we will look for current context from kubeconfig file
	if len(a.Server) == 0 {
		if defaultContext, defaultContextExists := a.StartingKubeConfig.Contexts[a.StartingKubeConfig.CurrentContext]; defaultContextExists {
			if cluster, exists := a.StartingKubeConfig.Clusters[defaultContext.Cluster]; exists {
				a.Server = cluster.Server
			}
		}
	}

	if err := a.GatherInfo(); err != nil {
		return err
	}

	_, err := a.SaveConfig()
	if err != nil {
		return err
	}

	log.Hint("You can create a new project by doing `odo project create <project-name>")
	log.Hint("You can use another project by doing `odo project set <project-name>")
	log.Hint("You can list existing projects by doing `odo project list")
	log.Hint("Look at `odo project --help` for other project related commands")

	return nil
}
