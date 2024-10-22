package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/dev6699/cube/task"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status command to list tasks.",
	Long: `cube status command.

The status command allows a user to get the status of tasks from the Cube manager.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := cmd.Flags().GetString("manager")
		if err != nil {
			return err
		}

		url := fmt.Sprintf("http://%s/tasks", manager)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		var tasks []*task.Task
		err = json.Unmarshal(body, &tasks)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "ID\tNAME\tCREATED\tSTATE\tCONTAINERNAME\tIMAGE\t")
		for _, task := range tasks {
			var start string
			if task.StartTime.IsZero() {
				start = fmt.Sprintf("%s ago", units.HumanDuration(time.Now().UTC().Sub(time.Now().UTC())))
			} else {
				start = fmt.Sprintf("%s ago", units.HumanDuration(time.Now().UTC().Sub(task.StartTime)))
			}
			state := task.State.String()
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t\n", task.ID.String(), task.Name, start, state, task.Name, task.Image)
		}

		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("manager", "m", "localhost:5555", "Manager to talk to")
}
