/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/collector"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/config"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/jsonmask"
	"reflect"
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		err := initialiseConf(cmd)
		if err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger := commandLogger(cmd)
		maskYaml, err := jsonmask.MaskYaml(collectorConf)
		if err != nil {
			logger.Errorf("%s", err)
			return
		}
		logger.Debugf("Configuration: %s", maskYaml)
		err = collector.Apply(logger, collectorConf)
		if err != nil {
			logger.Errorf("error: %s", err)
			return
		}
	},
}

func init() {
	configCmd.AddCommand(applyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// applyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// applyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

var collectorConf = &config.CollectorConf{}

const collectorConfFileName = "collector-conf"

func initialiseConf(cmd *cobra.Command) error {
	v := viper.New()

	v.SetConfigName(collectorConfFileName)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	err := v.Unmarshal(&collectorConf, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
				if f.Kind() != reflect.String || t != reflect.TypeOf(config.UnknownFormat) {
					return data, nil
				}
				var cf config.CoalesceFormat
				err := cf.Set(data.(string))
				if err != nil {
					return nil, err
				}
				return cf, nil
			},
		)))
	if err != nil {
		return err
	}

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	bindFlags(cmd, v)
	err = collectorConf.Validate()
	if err != nil {
		return err
	}
	return nil
}
