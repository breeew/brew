package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/starbx/brew-api/cmd/service"
)

func main() {
	root := &cobra.Command{
		Use:   "service",
		Short: "service",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("empty command")
		},
	}

	root.AddCommand(service.NewCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
