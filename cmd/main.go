package main

import (
	"fmt"
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
)

var log = logging.Logger("sample")

const version = "0.0.1"

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "list projects or nodes",
	Example: "list [cmd] /path/to/cofnig",
}

var deployCmd = &cobra.Command{
	Use:     "deploy",
	Short:   "deploy project",
	Example: "deploy --area=asia /path/to/config",
	Run: func(cmd *cobra.Command, args []string) {
		if err := deploy(cmd, args); err != nil {
			fmt.Println("deploy ", err.Error())
		}
	},
}

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "run sample",
	Example: "run /path/to/config",
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(cmd, args); err != nil {
			fmt.Println(err)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func execute() {
	// flags
	listRegionsCmd.Flags().String("area", "asia", "Specifying the area")

	deployCmd.Flags().String("area-id", "", "Specifying the area-id(scheduler) for project to deploy")
	deployCmd.Flags().String("region", "", "Specifying the region for project to deploy")
	deployCmd.Flags().String("name", "", "Specifying the project name")
	deployCmd.Flags().String("bundle-url", "", "Specifying the bundle url")
	deployCmd.Flags().String("nodes", "", "Specifying the nodes to deploy project")
	deployCmd.Flags().Int("replicas", 100, "Specifying the replicas")
	deployCmd.Flags().String("expiration", "", "Specifying the expiration")

	projectInfoCmd.Flags().String("project-id", "", "Specifying the project id")
	deleteProjectCmd.Flags().String("project-id", "", "Specifying the project id")

	listNodesCmd.Flags().String("area-id", "", "Specifying the area-id to list node")
	listNodesCmd.Flags().String("region", "", "Specifying the region to list node")
	listNodesCmd.Flags().Int("page", 0, "Specifying the page of list")
	listNodesCmd.Flags().Int("size", 20, "Specifying the size of page")

	listTunnelsCmd.Flags().String("project-id", "", "Specifying the project id")

	listProjectsCmd.Flags().Int("page", 0, "Specifying the page of list")
	listProjectsCmd.Flags().Int("size", 20, "Specifying the size of page")

	checkDelayCmd.Flags().Int("page", 0, "Specifying the page of list")
	checkDelayCmd.Flags().Int("size", 20, "Specifying the size of page")

	var rootCmd = &cobra.Command{}
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
	// rootCmd.AddCommand(listRegionsCmd)
	rootCmd.AddCommand(deployCmd)

	// projectCmd.AddCommand(listProjectsCmd, projectInfoCmd)
	// rootCmd.AddCommand(listProjectsCmd)
	listCmd.AddCommand(listProjectsCmd)
	listCmd.AddCommand(listNodesCmd)
	listCmd.AddCommand(listRegionsCmd)

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(projectInfoCmd)
	rootCmd.AddCommand(deleteProjectCmd)
	rootCmd.AddCommand(setNodeCmd)
	rootCmd.AddCommand(queryNodeCmd)
	rootCmd.AddCommand(listTunnelsCmd)
	rootCmd.AddCommand(checkDelayCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func main() {
	execute()
}
