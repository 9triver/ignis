package server

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type StartupHook func(config *Config) error

type Config struct {
	Head         HeadConfig
	Worker       WorkerConfig
	Addr         string
	Verbose      bool
	startupHooks []StartupHook
}

var config = &Config{}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Actor Platform for Cloud-Edge Applications (Server)",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Platform",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Actor Platform v%s\n", PlatformVersion)
	},
}

func init() {
	serverCmd.AddCommand(versionCmd)
	serverCmd.PersistentFlags().StringVarP(&config.Addr, "addr", "", "localhost", "Worker host address")
	serverCmd.PersistentFlags().BoolVarP(&config.Verbose, "verbose", "v", false, "Verbose output")
}

func Execute(hooks ...StartupHook) {
	config.startupHooks = hooks
	if err := serverCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
