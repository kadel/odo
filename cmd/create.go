package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/redhat-developer/odo/pkg/component"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	componentBinary string
	componentGit    string
	componentLocal  string
)

var componentCreateCmd = &cobra.Command{
	Use:   "create <component_type> [component_name] [flags]",
	Short: "Create new component",
	Long: `Create new component.
If component name is not provided, component type value will be used for name.
	`,
	Example: `  # Create new nodejs component with the source in current directory. 
  odo create nodejs

  # Create new nodejs component named 'frontend' with the source in './frontend' directory
  odo create nodejs frontend --local ./frontend

  # Create new nodejs component with source from remote git repository.
  odo create nodejs --git https://github.com/openshift/nodejs-ex.git
	`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugf("Component create called with args: %#v, flags: binary=%s, git=%s, local=%s", strings.Join(args, " "), componentBinary, componentGit, componentLocal)

		client := getOcClient()

		typeFlagsCount := 0
		// default source type is local
		sourceType := "local"
		if len(componentBinary) != 0 {
			typeFlagsCount++
			sourceType = "binary"
		}
		if len(componentLocal) != 0 {
			typeFlagsCount++
			sourceType = "local"
		}
		if len(componentGit) != 0 {
			typeFlagsCount++
			sourceType = "git"
		}

		if typeFlagsCount > 1 {
			fmt.Println("Only one of --git, --binary, --local may be specified.")
			os.Exit(1)
		}

		//We don't have to check it anymore, Args check made sure that args has at least one item
		// and no more than two
		componentType := args[0]
		componentName := args[0]
		if len(args) == 2 {
			componentName = args[1]
		}

		exists, err := component.Exists(client, componentName)
		if err != nil {
			checkError(err, "")
		}
		if exists {
			fmt.Printf("component with the name %s already exists in the current application\n", componentName)
			os.Exit(1)
		}

		switch sourceType {
		case "git":
			err := component.CreateFromGit(client, componentName, componentType, componentGit)
			checkError(err, "")
		case "local":
			// we want to use and save absolute path for component
			var dir string
			if len(componentLocal) > 0 {
				dir, err = filepath.Abs(componentLocal)
				checkError(err, "")
			} else {
				dir, err = filepath.Abs("./")
				checkError(err, "")
			}
			err = component.CreateFromPath(client, componentName, componentType, dir, false)
			checkError(err, "")
		case "binary":
			err = component.CreateFromPath(client, componentName, componentType, componentBinary, true)
			checkError(err, "")
		}

		// after component is successfully created, set is as active
		err = component.SetCurrent(client, componentName)
		checkError(err, "")
	},
}

func init() {
	componentCreateCmd.Flags().StringVar(&componentBinary, "binary", "", "Binary artifact")
	componentCreateCmd.Flags().StringVar(&componentGit, "git", "", "Git source")
	componentCreateCmd.Flags().StringVar(&componentLocal, "local", "", "Use local directory as a source for component")

	rootCmd.AddCommand(componentCreateCmd)
}
