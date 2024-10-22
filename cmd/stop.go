package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Args:  cobra.MinimumNArgs(1),
	Short: "Stop a running task.",
	Long: `cube stop command.

The stop command stops a running task.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := cmd.Flags().GetString("manager")
		if err != nil {
			return err
		}

		url := fmt.Sprintf("http://%s/tasks/%s", manager, args[0])
		client := &http.Client{}
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusNoContent {
			log.Printf("Error sending request: %v", err)
		}
		log.Printf("Task %v has been stopped.", args[0])

		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().StringP("manager", "m", "localhost:5555", "Manager to talk to")
}
