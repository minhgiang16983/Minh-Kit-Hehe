package command

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ImplementCmd returns the implement command
func ImplementCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "implement",
		Short: "Generate gRPC/REST handlers, interface, and boilerplate",
		Long:  `Generate gRPC/REST handlers, interface, and boilerplate from .proto files in the current base_service.`,
		Run: func(cmd *cobra.Command, args []string) {
			runImplement()
		},
	}
}

func runImplement() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ Failed to get current directory: %v", err)
	}
	serviceName := filepath.Base(cwd)
	if !isValidServiceName(serviceName) {
		log.Fatalf("❌ Invalid base_service name: %s", serviceName)
	}
	modulePrefix := toPascalCase(serviceName)
	protoDir := filepath.Join(cwd, "proto")
	interfaceFile := filepath.Join(cwd, "internal", "services", strings.ToLower(modulePrefix)+"_interface.go")
	handlerFile := filepath.Join(cwd, "internal", "services", strings.ToLower(modulePrefix)+"_handler.go")

	if err := generateInterfaceAndHandler(protoDir, interfaceFile, handlerFile, modulePrefix, serviceName); err != nil {
		log.Printf("⚠️ Interface generation failed: %v", err)
	} else {
		fmt.Println("✅ Interface & handler generated successfully")
	}
}
