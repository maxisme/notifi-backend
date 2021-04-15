package main

import (
	"fmt"
	"github.com/maxisme/notifi-backend/crypt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use: "crypt",
}

func main() {
	var hashCmd = &cobra.Command{
		Use:   "hash",
		Short: "Hash a string",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(crypt.Hash(args[0]))
		},
	}

	rootCmd.AddCommand(hashCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
