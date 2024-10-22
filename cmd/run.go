package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a new task.",
	Long: `cube run command.
The run command starts a new task.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := cmd.Flags().GetString("manager")
		if err != nil {
			return err
		}
		filename, err := cmd.Flags().GetString("filename")
		if err != nil {
			return err
		}

		if !fileExists(filename) {
			return fmt.Errorf("file %s does not exist", filename)
		}

		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		log.Printf("Data: %v\n", string(data))
		url := fmt.Sprintf("http://%s/tasks", manager)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusCreated {
			log.Printf("Error sending request: %v", resp.StatusCode)
		}
		defer resp.Body.Close()
		log.Println("Successfully sent task request to manager")

		return nil
	},
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !errors.Is(err, fs.ErrNotExist)
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("manager", "m", "localhost:5555", "Manager to talk to")
	runCmd.Flags().StringP("filename", "f", "task.json", "Task specification file")
}
