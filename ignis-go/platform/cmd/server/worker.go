package server

import (
	"actors/platform/actor/worker"
	"fmt"
	"os"

	console "github.com/asynkron/goconsole"
	"github.com/spf13/cobra"
)

type WorkerConfig struct {
	RemoteAddr string
	RemotePort int
}

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start a worker node",
	Run: func(cmd *cobra.Command, args []string) {
		if err := worker.Start(config.Addr, config.Worker.RemoteAddr, config.Worker.RemotePort); err != nil {
			fmt.Printf("Failed to start worker node: %s\n", err.Error())
			os.Exit(1)
		}
		for _, hook := range config.startupHooks {
			if err := hook(config); err != nil {
				fmt.Printf("Failed running startup hook: %s\n", err.Error())
				os.Exit(1)
			}
		}
		_, _ = console.ReadLine()
	},
}

func init() {
	workerCmd.Flags().StringVar(&config.Worker.RemoteAddr, "remote_addr", "", "Remote host address")
	workerCmd.Flags().IntVar(&config.Worker.RemotePort, "remote_port", 0, "Remote port")

	workerCmd.MarkFlagRequired("remoteAddr")
	workerCmd.MarkFlagRequired("remotePort")
	serverCmd.AddCommand(workerCmd)
}
