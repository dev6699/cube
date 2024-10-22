package cmd

import (
	"log"

	"github.com/dev6699/cube/manager"
	"github.com/spf13/cobra"
)

// managerCmd represents the manager command
var managerCmd = &cobra.Command{
	Use:   "manager",
	Short: "Manager command to operate a Cube manager",
	Long: `cube manager command.

The manager controls the orchestration system and is responsible for:
- Accepting tasks from users
- Scheduling tasks onto worker nodes
- Rescheduling tasks in the event of a node failure
- Periodically polling workers to get task updates`,
	RunE: func(cmd *cobra.Command, args []string) error {
		host, err := cmd.Flags().GetString("host")
		if err != nil {
			return err
		}
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}
		workers, err := cmd.Flags().GetStringSlice("workers")
		if err != nil {
			return err
		}
		scheduler, err := cmd.Flags().GetString("scheduler")
		if err != nil {
			return err
		}
		dbType, err := cmd.Flags().GetString("dbType")
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		m, err := manager.New(workers, scheduler, dbType)
		if err != nil {
			return err
		}

		api := manager.NewApi(host, port, m)
		go m.ProcessTasks(ctx)
		go m.UpdateTasks(ctx)
		go m.DoHealthChecks(ctx)
		go m.UpdateNodeStats(ctx)

		log.Printf("[manager] listening on http://%s:%d", host, port)
		return api.Start()
	},
}

func init() {
	rootCmd.AddCommand(managerCmd)
	managerCmd.Flags().StringP("host", "H", "0.0.0.0", "Hostname or IP address")
	managerCmd.Flags().IntP("port", "p", 5555, "Port on which to listen")
	managerCmd.Flags().StringSliceP("workers", "w", []string{"localhost:5556"}, "List of workers on which the manager will schedule tasks.")
	managerCmd.Flags().StringP("scheduler", "s", "roundrobin", "Nameof scheduler to use. (\"roundrobin\" or \"epvm\")")
	managerCmd.Flags().StringP("dbType", "d", "memory", "Type of datastore to use for events and tasks (\"memory\" or \"persistent\")")
}
