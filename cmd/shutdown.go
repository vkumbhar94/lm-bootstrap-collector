/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/collector"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/jsonmask"
)

// shutdownCmd represents the shutdown command
var shutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := initialise(cmd); err != nil {
			return err
		}
		err := conf.Validate()
		if err != nil {
			return fmt.Errorf("config validation failed with: %w", err)
		}
		if conf.Cleanup {
			return mustLMClient()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger := commandLogger(cmd)
		maskedJsonStr, err := jsonmask.MaskJson(conf)

		if err != nil {
			logger.Warn("Couldn't mask sensitive data of configuration, cannot printing configuration on stdout")
		} else {
			logger.Debugf("Configuration: %s", maskedJsonStr)
		}

		err = collector.Shutdown(logger, conf, lmClient)
		if err != nil {
			logger.Errorf("Shutdown failed with: %s", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(shutdownCmd)

	shutdownCmd.Flags().Int32Var(&conf.BackupCollectorID, "backup-collector-id", 0, "Backup Collector ID")
	shutdownCmd.Flags().VarP(&conf.Size, "collector-size", "", "Collector Size")
	shutdownCmd.Flags().BoolVar(&conf.Cleanup, "cleanup", false, "Cleanup")
	shutdownCmd.Flags().StringVar(&conf.Group, "collector-group", "", "Group")
	shutdownCmd.Flags().Int32Var(&conf.Version, "version", 0, "Version")
	shutdownCmd.Flags().StringVar(&conf.Description, "description", "", "Description")
	shutdownCmd.Flags().BoolVar(&conf.EnableFailBack, "enable-fail-back", false, "EnableFailBack")
	shutdownCmd.Flags().Int32Var(&conf.EscalatingChainID, "escalating-chain-id", 0, "EscalatingChainID")
	shutdownCmd.Flags().Int32Var(&conf.ID, "collector-id", 0, "ID")
	shutdownCmd.Flags().Int32Var(&conf.ResendInterval, "resend-interval", 0, "ResendInterval")
	shutdownCmd.Flags().BoolVar(&conf.SuppressAlertClear, "suppress-alert-clear", false, "SuppressAlertClear")
	shutdownCmd.Flags().BoolVar(&conf.UseEa, "use-ea", false, "UseEa")
	shutdownCmd.Flags().BoolVar(&conf.Kubernetes, "kubernetes", false, "Kubernetes")

	shutdownCmd.Flags().StringVar(&conf.IDS, "ids", "", "IDS")
	shutdownCmd.Flags().BoolVar(&conf.Debug, "debug", false, "Debug")
	shutdownCmd.Flags().IntVar(&conf.DebugIndex, "debug-index", 0, "Debug Index")

	_ = shutdownCmd.RegisterFlagCompletionFunc("collector-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"nano", "small", "medium", "large", "extra_large", "double_extra_large"}, cobra.ShellCompDirectiveDefault
	})
}
