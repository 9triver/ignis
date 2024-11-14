package server

import (
	"actors/platform/actor/head"
	"actors/platform/actor/http"
	"fmt"
	"os"

	console "github.com/asynkron/goconsole"
	"github.com/spf13/cobra"
)

type APIConfig struct {
	HTTPPort int
}

type HeadConfig struct {
	Port int
	API  APIConfig
}

var headCmd = &cobra.Command{
	Use:   "head",
	Short: "Start the head node",
	Run: func(cmd *cobra.Command, args []string) {
		if err := head.Start(config.Addr, config.Head.Port); err != nil {
			fmt.Printf("Failed to start head node: %s\n", err.Error())
			os.Exit(1)
		}

		http.Start(config.Head.API.HTTPPort)

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
	headCmd.Flags().IntVarP(&config.Head.Port, "port", "p", 2312, "Port to listen on")
	headCmd.Flags().IntVarP(&config.Head.API.HTTPPort, "http_port", "", 2313, "Port for http apis")
	serverCmd.AddCommand(headCmd)
}
