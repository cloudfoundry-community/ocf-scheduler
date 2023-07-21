package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/jobs"
)

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "sch",
	Short: "A CLI for interacting with scheduler",
	Long: `A CLI for interacting with scheduler

This top-level command is just a wrapper for other commands. Please see the
Available Commands section below.`,
}

func Execute() error {
	err := RootCmd.Execute()

	if err != nil {
		fmt.Println(err)
	}

	return err
}

func init() {
	RootCmd.AddCommand(calls.Command)
	RootCmd.AddCommand(jobs.Command)
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.SetConfigName(".sch")
	viper.AddConfigPath("$HOME")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
	}
}
