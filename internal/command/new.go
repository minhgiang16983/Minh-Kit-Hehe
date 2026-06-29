package command

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/minhgiang16983/Minh-Kit-Hehe/internal/template"
	"github.com/spf13/cobra"
)

// NewCmd returns the new base_service command
func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new [base_service-name]",
		Short: "Create a new base_service scaffold",
		Long:  `Create a new base_service scaffold with the given name, including all necessary files and structure.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			serviceName := args[0]
			generateNewService(serviceName, template.EmbeddedTemplates)
		},
	}
}

func generateNewService(serviceName string, templates embed.FS) {
	if !isValidServiceName(serviceName) {
		log.Fatalf("❌ Invalid base_service name: %s", serviceName)
	}
	servicePath := serviceName
	modulePath := "github.com/minhgiang16983/" + serviceName
	modulePrefix := toPascalCase(serviceName)

	fmt.Printf("📁 Creating base_service: %s\n", servicePath)

	err := os.MkdirAll(servicePath, os.ModePerm)
	if err != nil {
		log.Fatalf("❌ Failed to create base_service directory: %v", err)
	}

	err = fs.WalkDir(templates, "service-kit", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath := strings.TrimPrefix(path, "service-kit")
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == "" {
			return nil
		}
		targetPath := filepath.Join(servicePath, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, os.ModePerm)
		}

		targetPath = strings.TrimSuffix(targetPath, ".tpl")
		switch filepath.Base(targetPath) {
		case "_go.mod":
			targetPath = filepath.Join(filepath.Dir(targetPath), "go.mod")
		case "_go.sum":
			targetPath = filepath.Join(filepath.Dir(targetPath), "go.sum")
		}

		content, err := templates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		processed := replacePlaceholders(string(content), modulePath, modulePrefix, serviceName)
		if err := os.WriteFile(targetPath, []byte(processed), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}
		return nil
	})

	if err != nil {
		log.Fatalf("❌ Failed to generate project: %v", err)
	}
	fmt.Println("✅ Project generated successfully")

	protoDir := filepath.Join(servicePath, "proto")
	interfaceFile := filepath.Join(servicePath, "internal", "services", strings.ToLower(modulePrefix)+"_interface.go")
	handlerFile := filepath.Join(servicePath, "internal", "services", strings.ToLower(modulePrefix)+"_handler.go")

	if err := generateInterfaceAndHandler(protoDir, interfaceFile, handlerFile, modulePrefix, serviceName); err != nil {
		log.Printf("⚠️ Interface generation failed: %v", err)
	} else {
		fmt.Println("✅ Interface & handler generated successfully")
	}
}

func replacePlaceholders(content, modulePath, prefix, moduleName string) string {
	replacements := map[string]string{
		"github.com/minhgiang16983/service-kit": modulePath,
		"ServiceKit":                            prefix,
		"service-kit":                           moduleName,
	}
	for old, newVal := range replacements {
		content = strings.ReplaceAll(content, old, newVal)
	}
	return content
}
