package command

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// CleanCmd returns the clean command
func CleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Clean generated proto files",
		Long:  `Remove all generated proto files (pb directory and swagger docs).`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runClean(); err != nil {
				log.Fatalf("❌ Failed to clean generated files: %v", err)
			}
			fmt.Println("✅ Generated files cleaned successfully")
		},
	}
}

func runClean() error {
	// Remove pb directory
	if err := os.RemoveAll("pb"); err != nil {
		return fmt.Errorf("failed to remove pb directory: %w", err)
	}

	// Remove swagger files
	if err := os.RemoveAll("docs/swagger"); err != nil {
		return fmt.Errorf("failed to remove swagger directory: %w", err)
	}

	fmt.Println("🧹 Cleaned pb/ and docs/swagger/ directories")
	return nil
}
