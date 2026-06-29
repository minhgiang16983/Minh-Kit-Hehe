package command

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

const (
	protoDir     = "proto"
	swaggerDir   = "docs/swagger"
	defaultPbDir = "pb"
)

const (
	goLang     = "go"
	phpLang    = "php"
	jsLang     = "js"
	tsLang     = "ts"
	rubyLang   = "ruby"
	pythonLang = "python"
	javaLang   = "java"
	dartLang   = "dart"
	kotlinLang = "kotlin"
	swiftLang  = "swift"
	rustLang   = "rust"
)

var applyLanguages = []string{
	goLang,
	phpLang,
}

type GenClient struct {
	Lang                string
	Dir                 string
	PbDir               string
	WithResponseWrapper bool
	WithGw              bool
	WithSwagger         bool
	PluginDir           string
	ThirdPartyDir       string
}

// GenCmd returns the gen command
func GenCmd() *cobra.Command {
	var dir string
	var pbDir string
	var withResponseWrapper bool
	var withGrpcGateway bool
	var withSwagger bool
	var lang string
	var pluginDir string
	var thirdPartyDir string

	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate Go code and Swagger docs from .proto files",
		Long:  `Run protoc to generate Go code (pb, grpc, gateway) and Swagger documentation from .proto files.`,
		Run: func(cmd *cobra.Command, args []string) {

			if !slices.Contains(applyLanguages, strings.ToLower(lang)) {
				log.Fatalf("❌ Invalid language: %s", lang)
			}

			genClient := &GenClient{
				Lang:                lang,
				Dir:                 dir,
				PbDir:               pbDir,
				WithResponseWrapper: withResponseWrapper,
				WithGw:              withGrpcGateway,
				WithSwagger:         withSwagger,
				PluginDir:           pluginDir,
				ThirdPartyDir:       thirdPartyDir,
			}

			if err := genClient.runProtocGen(); err != nil {
				log.Fatalf("❌ Failed to generate proto code: %v", err)
			}
			fmt.Println("✅ Protobuf code and Swagger docs generated successfully")
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory to run protoc (default: current directory)")
	cmd.Flags().BoolVar(&withResponseWrapper, "with-response-wrapper", true, "Apply response wrapper to Swagger docs (default: true)")
	cmd.Flags().BoolVar(&withGrpcGateway, "with-gw", true, "Apply gateway grpc to proto (default: true)")
	cmd.Flags().BoolVar(&withSwagger, "with-swagger", true, "Apply gen swagger from proto (default: true)")
	cmd.Flags().StringVar(&pbDir, "pb-dir", defaultPbDir, "Directory to output proto code (default: pb)")
	cmd.Flags().StringVar(&lang, "lang", "go", "Language to generate (default: go)")
	cmd.Flags().StringVar(&pluginDir, "plugin-dir", "bin", "Directory to output proto code (default: bin)")
	cmd.Flags().StringVar(&thirdPartyDir, "third-party-dir", "proto/third_party", "Directory to output third party proto code (default: proto/third_party)")
	return cmd
}

func (s *GenClient) runProtocGen() error {

	currentProtoDir := filepath.Join(s.Dir, protoDir)
	// Check if protoc is available
	if _, err := os.Stat(currentProtoDir); os.IsNotExist(err) {
		return fmt.Errorf("proto directory not found")
	}

	// Create output directories if they don't exist

	pbDirectory := filepath.Join(s.Dir, s.PbDir)

	err := os.MkdirAll(pbDirectory, 0755)
	if err != nil {
		return fmt.Errorf("failed to create pb directory: %w", err)
	}

	// Swagger directory
	swDir := filepath.Join(s.Dir, swaggerDir)
	if s.WithSwagger {

		err = os.MkdirAll(swDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create docs/swagger directory: %w", err)
		}
	}

	currentDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	serviceDir := filepath.Join(currentDirectory, s.Dir)

	// Get all .proto files

	fmt.Println("scan proto files in ", path.Join(serviceDir, "proto", "*.proto"))
	protoFiles, err := filepath.Glob(path.Join(serviceDir, "proto", "*.proto"))
	if err != nil {
		return fmt.Errorf("failed to find .proto files: %w", err)
	}

	if len(protoFiles) == 0 {
		return fmt.Errorf("no .proto files found in proto directory")
	}

	// Gen third party
	if err = s.genAndRunThirdParty(); err != nil {
		return fmt.Errorf("failed to generate third party: %w", err)
	}

	// Build protoc command
	args := s.genArgsByLang(protoFiles)

	// Execute protoc command
	cmd := exec.Command("protoc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("🔧 Running: protoc %s\n", strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		return err
	}

	// Apply response wrapper if enabled
	if s.WithSwagger && s.WithResponseWrapper {
		fmt.Println("🔧 Applying response wrapper to Swagger docs...")
		if err := s.processSwaggerFiles(swDir); err != nil {
			return fmt.Errorf("failed to process swagger files: %w", err)
		}
		fmt.Println("✅ Response wrapper applied successfully")
	}

	return nil
}

func (s *GenClient) genArgsByLang(protoFiles []string) []string {
	switch s.Lang {
	case goLang:
		return s.genArgsGo(protoFiles)
	case phpLang:
		return s.genArgsPhp(protoFiles)
	}

	return []string{}
}

func (s *GenClient) genArgsGo(protoFiles []string) []string {
	var args []string

	currentDirectory, _ := os.Getwd()

	serviceDir := filepath.Join(currentDirectory, s.Dir)

	rootDir := filepath.Dir(serviceDir)
	argImports := []string{
		"-I=" + rootDir,
		"-I=" + protoDir,
		"-I=" + protoDir + "/third_party",
	}

	pbDirectory := filepath.Join(s.Dir, s.PbDir)
	// Go out
	argGoOut := []string{
		"--go_out=" + pbDirectory, "--go_opt=paths=import",
		"--go-grpc_out=" + pbDirectory, "--go-grpc_opt=paths=import",
	}

	// Gateway grpc
	argGrpcGatewayOut := []string{
		"--grpc-gateway_out=" + pbDirectory, "--grpc-gateway_opt=paths=import",
	}

	swDir := filepath.Join(s.Dir, swaggerDir)

	// swagger
	argsSwagger := []string{
		"--openapiv2_out=" + swDir, "--openapiv2_opt=json_names_for_fields=false",
		"--openapiv2_opt=allow_merge=true", "--openapiv2_opt=merge_file_name=doc",
		"--openapiv2_opt=disable_default_errors=true",
	}

	// validator
	argValidators := []string{
		"--validate_out=lang=go,paths=import:" + pbDirectory,
	}

	args = append(args, argImports...)
	args = append(args, argGoOut...)

	// If 'with gateway' is enable, gen gateway
	if s.WithGw {
		args = append(args, argGrpcGatewayOut...)
	}

	if s.WithSwagger {
		args = append(args, argsSwagger...)
	}

	args = append(args, argValidators...)

	args = append(args, protoFiles...)

	return args
}

func (s *GenClient) genAndRunThirdParty() error {
	if s.Lang != phpLang {
		return nil
	}

	var thirdPartyFiles []string

	// Process gen third party
	err := filepath.WalkDir(s.ThirdPartyDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".proto" {
			thirdPartyFiles = append(thirdPartyFiles, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println("gen third party files: ", thirdPartyFiles)

	pbDirectory := filepath.Join(s.Dir, s.PbDir)

	for _, file := range thirdPartyFiles {
		command := "protoc"
		args := []string{
			"-I=" + filepath.Dir(s.ThirdPartyDir),
			"-I=" + s.ThirdPartyDir,
			"--php_out=" + pbDirectory,
			file,
		}
		cmd := exec.Command(command, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (s *GenClient) genArgsPhp(protoFiles []string) []string {
	var args []string

	currentDirectory, _ := os.Getwd()

	serviceDir := filepath.Join(currentDirectory, s.Dir)

	rootDir := filepath.Dir(serviceDir)
	argImports := []string{
		"-I=" + rootDir,
		"-I=" + protoDir,
		"-I=" + protoDir + "/third_party",
	}

	pbDirectory := filepath.Join(s.Dir, s.PbDir)
	// Go out
	argGoOut := []string{
		"--php_out=" + pbDirectory,
		"--grpc_out=" + pbDirectory,
	}

	pluginPaths := filepath.Join(rootDir, "bin", "grpc_php_plugin")

	// Plugin
	argsPlugin := []string{
		"--plugin=protoc-gen-grpc=" + pluginPaths,
	}

	swDir := filepath.Join(s.Dir, swaggerDir)

	// swagger
	argsSwagger := []string{
		"--openapiv2_out=" + swDir, "--openapiv2_opt=json_names_for_fields=false",
		"--openapiv2_opt=allow_merge=true", "--openapiv2_opt=merge_file_name=doc",
		"--openapiv2_opt=disable_default_errors=true",
	}

	args = append(args, argImports...)
	args = append(args, argGoOut...)

	// If 'with gateway' is enable, gen gateway
	args = append(args, argsPlugin...)

	if s.WithSwagger {
		args = append(args, argsSwagger...)
	}

	args = append(args, protoFiles...)

	return args
}

// processSwaggerFiles processes all Swagger files in a directory to wrap responses
func (s *GenClient) processSwaggerFiles(swaggerDir string) error {
	// Find all .swagger.json files
	var swaggerFiles []string
	err := filepath.WalkDir(swaggerDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".swagger.json") {
			swaggerFiles = append(swaggerFiles, path)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory %s: %w", swaggerDir, err)
	}

	if len(swaggerFiles) == 0 {
		fmt.Printf("No Swagger files found in %s\n", swaggerDir)
		return nil
	}

	fmt.Printf("Found %d Swagger files to process:\n", len(swaggerFiles))
	for _, file := range swaggerFiles {
		fmt.Printf("  - %s\n", filepath.Base(file))
	}

	// Process each file
	for _, inputFile := range swaggerFiles {
		if err := s.processSwaggerFile(inputFile); err != nil {
			return fmt.Errorf("error processing %s: %w", inputFile, err)
		}
	}

	return nil
}

// processSwaggerFile processes a Swagger file to wrap all response schemas
func (s *GenClient) processSwaggerFile(inputFile string) error {
	// Read input file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", inputFile, err)
	}

	// Parse JSON
	var swagger map[string]interface{}
	if err := json.Unmarshal(data, &swagger); err != nil {
		return fmt.Errorf("failed to parse JSON from %s: %w", inputFile, err)
	}

	// Process all response schemas
	paths, ok := swagger["paths"].(map[string]interface{})
	if ok {
		for _, pathItem := range paths {
			if pathMap, ok := pathItem.(map[string]interface{}); ok {
				for method, operation := range pathMap {
					if strings.ToLower(method) == "get" || strings.ToLower(method) == "post" ||
						strings.ToLower(method) == "put" || strings.ToLower(method) == "delete" ||
						strings.ToLower(method) == "patch" {

						if opMap, ok := operation.(map[string]interface{}); ok {
							if responses, ok := opMap["responses"].(map[string]interface{}); ok {
								for _, response := range responses {
									if respMap, ok := response.(map[string]interface{}); ok {
										if schema, exists := respMap["schema"]; exists {
											respMap["schema"] = wrapResponseSchema(schema)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Add standard response definitions
	if swagger["definitions"] == nil {
		swagger["definitions"] = make(map[string]interface{})
	}

	definitions := swagger["definitions"].(map[string]interface{})

	definitions["ResponseStatus"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"error_code": map[string]interface{}{
				"type":        "integer",
				"description": "HTTP status code",
			},
			"error_message": map[string]interface{}{
				"type":        "string",
				"description": "Human-readable error message",
			},
			"alert_message": map[string]interface{}{
				"type":        "string",
				"description": "Alert message for UI/UX",
			},
		},
	}

	definitions["StandardResponse"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"$ref": "#/definitions/ResponseStatus",
			},
			"data": map[string]interface{}{
				"type":        "object",
				"description": "Response data",
			},
		},
	}

	// Write processed file
	outputData, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(inputFile, outputData, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", inputFile, err)
	}

	fmt.Printf("Updated %s\n", inputFile)
	return nil
}

// wrapResponseSchema wraps a schema in standard response format
func wrapResponseSchema(schema interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"error_code": map[string]interface{}{
						"type":        "integer",
						"description": "HTTP status code",
					},
					"error_message": map[string]interface{}{
						"type":        "string",
						"description": "Human-readable error message",
					},
					"alert_message": map[string]interface{}{
						"type":        "string",
						"description": "Alert message for UI/UX",
					},
				},
			},
			"data": schema,
		},
	}
}
