/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/client/logicmonitor"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/collector"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const envPrefix = "COLLECTOR"
const configFileName = "collector"

var conf = &config.Config{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lm-bootstrap-collector",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		err := initialise(cmd)
		if err != nil {
			return err
		}
		err = conf.Validate()
		if err != nil {
			return fmt.Errorf("config validation failed with: %w", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		logrus.SetOutput(cmd.OutOrStdout())
		logrus.SetReportCaller(true)
		logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})
		logger := logrus.WithField("command", cmd.Name())
		b, err := json.Marshal(conf)
		if err != nil {
			logger.Error("Failed to marshal config")
			return
		}
		logger.Infof("Config: %s", b)

		client, err := logicmonitor.NewLMClient(conf)
		if err != nil {
			logger.Error("Cannot create logicmonitor client")
			return
		}
		if err := collector.Start(logger, conf, client); err != nil {
			logger.Infof("Install failed with: %s", err)
			return
		}
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.lm-bootstrap-collector.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().StringVar(&conf.Account, "account", "", "Logicmonitor Account")
	rootCmd.Flags().StringVar(&conf.AccessID, "access-id", "", "Logicmonitor Access ID")
	rootCmd.Flags().StringVar(&conf.AccessKey, "access-key", "", "Logicmonitor Access Key")
	rootCmd.Flags().Int32Var(&conf.BackupCollectorID, "backup-collector-id", 0, "Backup Collector ID")
	rootCmd.Flags().VarP(&conf.CollectorSize, "collector-size", "", "Collector Size")
	rootCmd.Flags().BoolVar(&conf.Cleanup, "cleanup", false, "Cleanup")
	rootCmd.Flags().StringVar(&conf.CollectorGroup, "collector-group", "", "CollectorGroup")
	rootCmd.Flags().Int32Var(&conf.Version, "version", 0, "Version")
	rootCmd.Flags().StringVar(&conf.Description, "description", "", "Description")
	rootCmd.Flags().BoolVar(&conf.EnableFailBack, "enable-fail-back", false, "EnableFailBack")
	rootCmd.Flags().Int32Var(&conf.EscalatingChainID, "escalating-chain-id", 0, "EscalatingChainID")
	rootCmd.Flags().Int32Var(&conf.CollectorID, "collector-id", 0, "CollectorID")
	rootCmd.Flags().Int32Var(&conf.ResendInterval, "resend-interval", 0, "ResendInterval")
	rootCmd.Flags().BoolVar(&conf.SuppressAlertClear, "suppress-alert-clear", false, "SuppressAlertClear")
	rootCmd.Flags().BoolVar(&conf.UseEa, "use-ea", false, "UseEa")
	rootCmd.Flags().BoolVar(&conf.Kubernetes, "kubernetes", false, "Kubernetes")
	rootCmd.Flags().StringVar(&conf.ProxyUrl, "proxy-url", "", "ProxyUrl")
	rootCmd.Flags().StringVar(&conf.ProxyUser, "proxy-user", "", "ProxyUser")
	rootCmd.Flags().StringVar(&conf.ProxyPass, "proxy-pass", "", "ProxyPass")
	rootCmd.Flags().BoolVar(&conf.IgnoreSSL, "ignore-ssl", false, "IgnoreSSL")
	rootCmd.Flags().StringVar(&conf.IDS, "ids", "", "IDS")
	rootCmd.Flags().BoolVar(&conf.Debug, "debug", false, "Debug")
	rootCmd.Flags().IntVar(&conf.DebugIndex, "debug-index", 0, "Debug Index")

	_ = rootCmd.RegisterFlagCompletionFunc("collector-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"nano", "small", "medium", "large", "extra_large", "double_extra_large"}, cobra.ShellCompDirectiveDefault
	})
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

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		} else if !f.Changed {
			key := strings.ToLower(strings.ReplaceAll(f.Name, "-", ""))
			if v.IsSet(key) {
				val := v.Get(key)
				cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			}
		}

	})
}
