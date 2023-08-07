package main

import (
	"fmt"
	"os"

	"github.com/cjp2600/trep/cmd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "trep",
	Short: "trep is a CLI for running and formatting output",
	Long:  `trep is a CLI tool developed for running commands and formatting their output in a more readable and colorful way.`,
}

func main() {
	rootCmd.AddCommand(cmd.ExecCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
