package main

import (
	"encoding/json"
	"fmt"
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
)

var log = logging.Logger("sample")

const version = "0.0.1"

var listNodesCmd = &cobra.Command{
	Use:     "node",
	Short:   "list all nodes",
	Example: "node /path/to/config",
	Run: func(cmd *cobra.Command, args []string) {
		pInfos, err := listNodes(cmd, args)
		if err != nil {
			fmt.Println("list projects ", err.Error())
			return
		}

		if len(pInfos) == 0 {
			fmt.Println("no node exist")
			return
		}

		for _, info := range pInfos {
			fmt.Printf("%s %s %s\n", info.Name, info.AreaID, info.ID)
			for _, ap := range info.Nodes {
				status := "failed"
				if ap.Status == 1 {
					status = "servicing"
				}
				fmt.Printf("%s %s %s %s\n", ap.AreaID, ap.ID, ap.IP, status)
			}
		}

	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "list projects or nodes",
	Example: "list [cmd] /path/to/cofnig",
}

var listProjectsCmd = &cobra.Command{
	Use:     "project",
	Short:   "list all projects",
	Example: "project /path/to/config",
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := listProjects(cmd, args)
		if err != nil {
			fmt.Println("list projects ", err.Error())
			return
		}

		if len(projects) == 0 {
			fmt.Println("no project exist")
			return
		}

		for _, project := range projects {
			fmt.Printf("%s %s %d %s %s %s\n", project.ID, project.Status, project.Replicas, project.Name, project.AreaID, project.Region)

		}

	},
}

var projectInfoCmd = &cobra.Command{
	Use:     "project",
	Short:   "get project info",
	Example: "project --project-id=your-project-id /path/to/config",
	Run: func(cmd *cobra.Command, args []string) {
		projectInfo, err := getProjectInfo(cmd, args)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Println("Project ID: ", projectInfo.ID)
		for _, accessPoint := range projectInfo.Nodes {
			fmt.Printf("%s %s\n", accessPoint.ID, accessPoint.URL)
		}

	},
}

var deleteProjectCmd = &cobra.Command{
	Use:     "delete",
	Short:   "delete project",
	Example: "delete --project-id=your-project-id /path/to/config",
	Run: func(cmd *cobra.Command, args []string) {
		err := deleteProjectInfo(cmd, args)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		projectID, _ := cmd.Flags().GetString("project-id")
		fmt.Printf("delete %s success\n", projectID)
	},
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

var listRegionsCmd = &cobra.Command{
	Use:     "regions",
	Short:   "list regions",
	Example: "regions --area=asia /path/to/config",
	Run: func(cmd *cobra.Command, args []string) {
		areaList, err := getRegions(cmd, args)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, area := range areaList.List {
			regions, _ := json.Marshal(area.Regions)
			fmt.Printf("area-id: %s regions: %s\n", area.AreaID, string(regions))
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

	var rootCmd = &cobra.Command{}
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listRegionsCmd)
	rootCmd.AddCommand(deployCmd)

	// projectCmd.AddCommand(listProjectsCmd, projectInfoCmd)
	// rootCmd.AddCommand(listProjectsCmd)
	listCmd.AddCommand(listProjectsCmd)
	listCmd.AddCommand(listNodesCmd)

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(projectInfoCmd)
	rootCmd.AddCommand(deleteProjectCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func main() {
	execute()
}
