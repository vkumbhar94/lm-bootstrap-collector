/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/json"
	"time"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

type versionMap struct {
	Version      string
	Commit       string
	BuildDateUTC string
	BuildDate    string
	BuiltBy      string
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:               "version",
	ValidArgsFunction: cobra.NoFileCompletions,
	Short:             "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if Short {
			cmd.Printf("%s\n", Version)
			return
		}
		vm := NewVersionMap()
		marshal, err := json.Marshal(vm) // nolint: errcheck
		if err != nil {
			cmd.Printf("failing to print version info: %s\n", err)
			return
		}
		cmd.Println(string(marshal))
	},
}

func NewVersionMap() *versionMap {
	vm := &versionMap{
		Version:      Version,
		Commit:       Commit,
		BuiltBy:      BuiltBy,
		BuildDateUTC: "unknown",
		BuildDate:    "unknown",
	}
	date, err := time.Parse(time.RFC3339, Date) // nolint: errcheck
	if err != nil {
		return vm
	}
	vm.BuildDateUTC = date.UTC().String()
	vm.BuildDate = date.Local().String()

	return vm
}

var Short bool

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	versionCmd.Flags().BoolVar(&Short, "short", false, "Show short Version")
}
