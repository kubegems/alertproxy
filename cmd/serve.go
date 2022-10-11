/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"kubegems.io/alertproxy/config"
	"kubegems.io/alertproxy/pkg/server"
	"sigs.k8s.io/yaml"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer cancel()
		opts := config.ProxyConfigs{}
		bts, err := os.ReadFile(cfgFile)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(bts, &opts); err != nil {
			return err
		}
		return server.Run(ctx, &opts)
	},
}

var cfgFile string

func init() {
	rootCmd.AddCommand(serveCmd)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config/deploy/alertproxy.yaml", "config file (default is config/deploy/alertproxy.yaml)")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
