package cmd

import (
	"fmt"
	"log"

	"github.com/dev6699/cube/worker"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Worker command to operate a Cube worker node.",
	Long: `cube worker command.

The worker runs tasks and responds to the manager's requests about task state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		host, err := cmd.Flags().GetString("host")
		if err != nil {
			return err
		}
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		dbType, err := cmd.Flags().GetString("dbtype")
		if err != nil {
			return err
		}

		w, err := worker.New(name, dbType)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		api := worker.NewApi(host, port, w)
		go w.RunTasks(ctx)
		go w.CollectStats(ctx)
		go w.UpdateTasks(ctx)

		log.Printf("[worker] listening on http://%s:%d\n", host, port)
		return api.Start()
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
	workerCmd.Flags().StringP("host", "H", "0.0.0.0", "Hostname or IP address")
	workerCmd.Flags().IntP("port", "p", 5556, "Port on which to listen")
	workerCmd.Flags().StringP("name", "n", fmt.Sprintf("worker-%s", uuid.New().String()), "Name of the worker")
	workerCmd.Flags().StringP("dbtype", "d", "memory", "Type of datastore to use for tasks (\"memory\" or \"bolt\")")
}
