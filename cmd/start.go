/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/collector"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/jsonmask"
)

const (
	envPrefix      = "COLLECTOR"
	configFileName = "collector"
)

var conf = &config.Config{Size: config.Small}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
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
		return mustLMClient()
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger := commandLogger(cmd)
		maskedJsonStr, err := jsonmask.MaskJson(conf)

		if err != nil {
			logger.Warn("Couldn't mask sensitive data of configuration, cannot printing configuration on stdout")
		} else {
			logger.Debugf("Configuration: %s", maskedJsonStr)
		}

		if err := collector.Start(logger, creds, conf, lmClient); err != nil {
			logger.Infof("Install failed with: %s", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.lm-bootstrap-collector.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	startCmd.Flags().Int32Var(&conf.BackupCollectorID, "backup-collector-id", 0, "Backup Collector ID")
	startCmd.Flags().VarP(&conf.Size, "size", "", "Collector Size")
	startCmd.Flags().BoolVar(&conf.Cleanup, "cleanup", false, "Cleanup")
	startCmd.Flags().StringVar(&conf.Group, "group", "", "Group")
	startCmd.Flags().Int32Var(&conf.Version, "version", 0, "Version")
	startCmd.Flags().StringVar(&conf.Description, "description", "", "Description")
	startCmd.Flags().BoolVar(&conf.EnableFailBack, "enable-fail-back", false, "EnableFailBack")
	startCmd.Flags().Int32Var(&conf.EscalatingChainID, "escalating-chain-id", 0, "EscalatingChainID")
	startCmd.Flags().Int32Var(&conf.ID, "id", 0, "ID")
	startCmd.Flags().Int32Var(&conf.ResendInterval, "resend-interval", 0, "ResendInterval")
	startCmd.Flags().BoolVar(&conf.SuppressAlertClear, "suppress-alert-clear", false, "SuppressAlertClear")
	startCmd.Flags().BoolVar(&conf.UseEa, "use-ea", false, "UseEa")
	startCmd.Flags().BoolVar(&conf.Kubernetes, "kubernetes", false, "Kubernetes")

	startCmd.Flags().StringVar(&conf.IDS, "ids", "", "IDS")
	startCmd.Flags().BoolVar(&conf.Debug, "debug", false, "Debug")
	startCmd.Flags().IntVar(&conf.DebugIndex, "debug-index", 0, "Debug Index")
	startCmd.Flags().BoolVar(&conf.SkipInstall, "skip-install", false, "Skip Install (only download)")
	startCmd.Flags().BoolVar(&conf.RunAsSudo, "run-as-sudo", false, "Run As Sudo")
	startCmd.Flags().StringVar(&conf.InstallUser, "install-user", "logicmonitor", "Install User")

	_ = startCmd.RegisterFlagCompletionFunc("size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"nano", "small", "medium", "large", "extra_large", "double_extra_large"}, cobra.ShellCompDirectiveDefault
	})
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initialise(cmd *cobra.Command) error {
	v := viper.New()

	v.SetConfigName(configFileName)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	bindFlags(cmd, v)
	return nil
}
