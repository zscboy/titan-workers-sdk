package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	worker "github.com/zscboy/titan-workers-sdk"
	"github.com/zscboy/titan-workers-sdk/config"
	"github.com/zscboy/titan-workers-sdk/tablewriter"
)

func listProjects(cmd *cobra.Command, args []string) ([]*worker.Project, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("Please specify the name of the config file")
	}

	configFilePath := args[0]
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist.", configFilePath)
	}

	cfg, err := config.ParseConfig(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("parse config error " + err.Error())
	}

	wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return nil, fmt.Errorf("NewWorker %s", err.Error())
	}

	return w.GetProjects()

}

func getProjectInfo(cmd *cobra.Command, args []string) (*worker.PorjectInfo, error) {
	projectID, err := cmd.Flags().GetString("project-id")
	if len(projectID) == 0 || err != nil {
		return nil, fmt.Errorf("Must set --project-id")
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("Please specify the name of the config file")
	}

	configFilePath := args[0]
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist.", configFilePath)
	}

	cfg, err := config.ParseConfig(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("parse config error " + err.Error())
	}

	wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return nil, fmt.Errorf("NewWorker %s", err.Error())
	}

	return w.GetProjectInfo(projectID)
}

func deleteProjectInfo(cmd *cobra.Command, args []string) error {
	projectID, err := cmd.Flags().GetString("project-id")
	if len(projectID) == 0 || err != nil {
		return fmt.Errorf("Must set --project-id")
	}

	if len(args) == 0 {
		return fmt.Errorf("Please specify the name of the config file")
	}

	configFilePath := args[0]
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist.", configFilePath)
	}

	cfg, err := config.ParseConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("parse config error " + err.Error())
	}

	wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return fmt.Errorf("NewWorker %s", err.Error())
	}

	return w.DeleteProject(projectID)
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

		tw := tablewriter.New(
			tablewriter.Col("ProjectID"),
			tablewriter.Col("Name"),
			tablewriter.Col("Status"),
			tablewriter.Col("Replicas"),
			tablewriter.Col("AreaID"),
			tablewriter.Col("Region"),
		)

		for _, project := range projects {
			m := map[string]interface{}{
				"ProjectID": project.ID,
				"Name":      project.Name,
				"Status":    project.Status,
				"Replicas":  project.Replicas,
				"AreaID":    project.AreaID,
				"Region":    project.Region,
			}

			tw.Write(m)

		}

		tw.Flush(os.Stdout)
		fmt.Printf(color.YellowString("\nTotal: %d ", len(projects)))
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
