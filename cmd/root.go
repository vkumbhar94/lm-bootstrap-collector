/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/logicmonitor/lm-sdk-go/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/client/logicmonitor"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/jsonmask"
)

var (
	lmClient *client.LMSdkGo = nil
	creds                    = &config.Creds{}
)

var ExemptCredsCmds = map[string]struct{}{
	"lm-bootstrap-collector.version":      {},
	"lm-bootstrap-collector.config.apply": {},
}
var logLevel = LogLevel(logrus.InfoLevel)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lm-bootstrap-collector",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := initialise(cmd); err != nil {
			return err
		}
		err := creds.Validate()
		if err != nil {
			if _, ok := ExemptCredsCmds[getCmdFqnm(cmd)]; !ok {
				return fmt.Errorf("credentials validation failed with: %w", err)
			}
		}

		logrus.SetOutput(cmd.OutOrStdout())
		if logrus.Level(logLevel) > logrus.DebugLevel {
			logrus.SetReportCaller(true)
		}
		logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})
		logrus.SetLevel(logrus.Level(logLevel))
		logger := commandLogger(cmd)

		maskedJsonStr, err := jsonmask.MaskJson(creds, "AccessID", "AccessKey", "ProxyPass", "SudoPass")

		if err != nil {
			logger.Warn("Couldn't mask sensitive data of configuration, cannot printing credentials on stdout")
		} else {
			logger.Debugf("Credentials: %s", maskedJsonStr)
		}

		if creds.Account != "" && creds.AccessID != "" && creds.AccessKey != "" {
			var err error
			lmClient, err = logicmonitor.NewLMClient(creds)
			if err != nil {
				logger.Debugf("Logicmonitor client creation failed with: %s", err)
			}
		}
		return nil
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

func commandLogger(cmd *cobra.Command) logrus.FieldLogger {
	return logrus.WithField("command", getCmdFqnm(cmd))
}

func getCmdFqnm(cmd *cobra.Command) string {
	if cmd.Parent() == nil {
		return cmd.Name()
	}
	return fmt.Sprintf("%s.%s", getCmdFqnm(cmd.Parent()), cmd.Name())
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
	rootCmd.PersistentFlags().Var(&logLevel, "log-level", "Log Level")
	rootCmd.PersistentFlags().StringVar(&creds.Account, "account", "", "Logicmonitor Account")
	rootCmd.PersistentFlags().StringVar(&creds.AccessID, "access-id", "", "Logicmonitor Access ID")
	rootCmd.PersistentFlags().StringVar(&creds.AccessKey, "access-key", "", "Logicmonitor Access Key")
	rootCmd.PersistentFlags().StringVar(&creds.ProxyUrl, "proxy-url", "", "ProxyUrl")
	rootCmd.PersistentFlags().StringVar(&creds.ProxyUser, "proxy-user", "", "ProxyUser")
	rootCmd.PersistentFlags().StringVar(&creds.ProxyPass, "proxy-pass", "", "ProxyPass")
	rootCmd.PersistentFlags().BoolVar(&creds.IgnoreSSL, "ignore-ssl", false, "IgnoreSSL")
	rootCmd.PersistentFlags().StringVar(&creds.SudoPass, "sudo-pass", "", "Sudo Password")

	_ = rootCmd.RegisterFlagCompletionFunc("log-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"info", "debug", "trace", "warn", "warning", "error", "fatal", "panic"}, cobra.ShellCompDirectiveDefault
	})
}

type LogLevel logrus.Level

func (ll *LogLevel) Set(v string) error {
	level, err := logrus.ParseLevel(v)
	if err != nil {
		return err
	}
	*ll = LogLevel(level)
	return nil
}

func (ll *LogLevel) Type() string {
	return "logrus.LogLevel"
}

func (ll *LogLevel) String() string {
	return logrus.Level(*ll).String()
}

func mustLMClient() error {
	if lmClient == nil {
		return fmt.Errorf("logicmonitor client couldn't create, check credentials")
	}
	return nil
}
