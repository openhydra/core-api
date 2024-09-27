package app

import (
	"core-api/cmd/core-api-server/app/option"
	"fmt"

	"core-api/pkg/core/apiserver"
	coreApiLog "core-api/pkg/logger"

	"github.com/common-nighthawk/go-figure"
	"github.com/spf13/cobra"
)

func NewCommand(version string) *cobra.Command {
	option := &option.Option{}
	cmd := &cobra.Command{
		Use:     "core-api-server",
		Long:    "core-api-server is a server daemon that provides a RESTful API for core-api",
		Example: figure.NewColorFigure("core-api", "isometric1", "green", true).String(),
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Print version and exit",
		Long:    "version subcommand will print version and exit",
		Example: "core-api-server version",
		Run: func(_ *cobra.Command, args []string) {
			fmt.Println("version:", version)
		},
	}

	runCommand := &cobra.Command{
		Use:     "run",
		Short:   "Run core-api-server",
		Long:    "Run core-api-server",
		Example: "core-api-server --config /etc/core-api-server-config.yaml run",
		Run: func(_ *cobra.Command, args []string) {
			config, err := option.GenerateConfig(true)
			if err != nil {
				fmt.Println("Failed to generate config", err)
				return
			}

			// set idflag version to config
			config.CoreApiConfig.GitVersion = version
			// init logger before use it
			// we can config log level in config file
			coreApiLog.InitLogger(config.CoreApiConfig.LogLevel)

			err = apiserver.RunServer(config)
			if err != nil {
				fmt.Println("Failed to run server")
				return
			}
		},
	}

	option.BindFlags(runCommand.Flags())

	cmd.AddCommand(versionCmd)
	cmd.AddCommand(runCommand)
	return cmd
}
