package main

import (
	"fmt"
	"os"

	"github.com/minhgiang16983/Minh-Kit-Hehe/internal/command"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "minh-kit",
		Short: "Minh Kit - A Go base_service development toolkit",
		Long: `Minh Kit is a comprehensive toolkit for generating Go microservices with gRPC/REST support.
It provides templates and generators for creating new services, implementing interfaces, and generating models from SQL schemas.`,
	}

	// Add commands
	rootCmd.AddCommand(command.ImplementCmd())
	rootCmd.AddCommand(command.ModelCmd())
	rootCmd.AddCommand(command.NewCmd())
	rootCmd.AddCommand(command.GenCmd())
	rootCmd.AddCommand(command.CleanCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}
