package command

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// isValidServiceName checks if the base_service name is valid
func isValidServiceName(name string) bool {
	// Check valid name: only contains letters, numbers, and hyphens
	return regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(name) && !strings.HasPrefix(name, "-") && !strings.HasSuffix(name, "-")
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "-")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	return strings.Join(parts, "")
}

// extractExistingRPCs extracts existing RPC signatures from an interface file
func extractExistingRPCs(ifacePath string) (map[string]bool, error) {
	existingRPCs := make(map[string]bool)
	if _, err := os.Stat(ifacePath); os.IsNotExist(err) {
		return existingRPCs, nil
	}

	data, err := os.ReadFile(ifacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read interface file %s: %w", ifacePath, err)
	}

	// Find methods in interface
	interfaceRegex := regexp.MustCompile(`\s*(\w+)\(ctx context\.Context, req \*\w+\.\w+\) \(\*\w+\.\w+, error\)\s*`)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := interfaceRegex.FindStringSubmatch(line)
		if len(matches) == 2 {
			existingRPCs[matches[1]] = true
		}
	}
	return existingRPCs, nil
}

// generateInterfaceAndHandler generates interface and handler files from .proto files
func generateInterfaceAndHandler(protoDir, ifacePath, handlerPath, prefix string, serviceName string) error {
	moduleName := serviceName

	protoFiles, err := os.ReadDir(protoDir)
	if err != nil {
		return fmt.Errorf("failed to read proto directory: %w", err)
	}

	var rpcs []string
	var rpcDetails []struct {
		Method string
		Req    string
		Resp   string
	}

	for _, file := range protoFiles {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".proto") {
			continue
		}
		fullPath := filepath.Join(protoDir, file.Name())
		parsed, details, err := extractRPCsFromProto(fullPath)
		if err != nil {
			log.Printf("⚠️ Skipped %s: %v", fullPath, err)
			continue
		}
		rpcs = append(rpcs, parsed...)
		rpcDetails = append(rpcDetails, details...)
	}

	if len(rpcs) == 0 {
		return fmt.Errorf("no RPCs found")
	}

	// INTERFACE
	existingRPCs, _ := extractExistingRPCs(ifacePath)

	// Filter out existing RPCs
	var newRPCs []struct {
		Method string
		Req    string
		Resp   string
	}
	for _, rpc := range rpcDetails {
		if !existingRPCs[rpc.Method] {
			newRPCs = append(newRPCs, rpc)
		}
	}

	// Generate interface using template
	interfaceTemplate := `package services

import (
	"context"
	pb "github.com/minhgiang16983/{{.ModuleName}}/pb"
)

// {{.Prefix}}ServiceInterface defines the interface for {{.Prefix}}Service
type {{.Prefix}}ServiceInterface interface {
{{- range .RPCs}}
	{{.Method}}(ctx context.Context, req *pb.{{.Req}}) (*pb.{{.Resp}}, error)
{{- end}}
}
`

	tmpl, err := template.New("interface").Parse(interfaceTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse interface template: %w", err)
	}

	var ib bytes.Buffer
	err = tmpl.Execute(&ib, struct {
		ModuleName string
		Prefix     string
		RPCs       []struct {
			Method string
			Req    string
			Resp   string
		}
	}{
		ModuleName: moduleName,
		Prefix:     prefix,
		RPCs:       newRPCs,
	})
	if err != nil {
		return fmt.Errorf("failed to execute interface template: %w", err)
	}

	if err := os.WriteFile(ifacePath, ib.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write interface file: %w", err)
	}

	// HANDLER
	existingHandler, _ := os.ReadFile(handlerPath)
	existingHandlerMethods := make(map[string]bool)
	if len(existingHandler) > 0 {
		re := regexp.MustCompile(fmt.Sprintf(`func $begin:math:text$s \\*%sService$end:math:text$ (\w+)\(`, prefix))
		matches := re.FindAllStringSubmatch(string(existingHandler), -1)
		for _, match := range matches {
			if len(match) > 1 {
				existingHandlerMethods[match[1]] = true
			}
		}
	}

	// Filter out existing handler methods
	var newHandlerRPCs []struct {
		Method string
		Req    string
		Resp   string
	}
	for _, rpc := range rpcDetails {
		if !existingHandlerMethods[rpc.Method] {
			newHandlerRPCs = append(newHandlerRPCs, rpc)
		}
	}

	var hb bytes.Buffer

	// If no existing handler, write package and imports
	if len(existingHandler) == 0 {
		handlerHeaderTemplate := `package services

import (
	"context"
	"fmt"
	pb "github.com/minhgiang16983/{{.ModuleName}}/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

`
		headerTmpl, err := template.New("handlerHeader").Parse(handlerHeaderTemplate)
		if err != nil {
			return fmt.Errorf("failed to parse handler header template: %w", err)
		}

		err = headerTmpl.Execute(&hb, struct {
			ModuleName string
		}{
			ModuleName: moduleName,
		})
		if err != nil {
			return fmt.Errorf("failed to execute handler header template: %w", err)
		}
	} else {
		// Write existing content
		hb.Write(existingHandler)
	}

	// Generate new handler methods using template
	handlerMethodTemplate := `// {{.Method}} is not implemented yet
func (s *{{.Prefix}}Service) {{.Method}}(ctx context.Context, req *pb.{{.Req}}) (*pb.{{.Resp}}, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}
	return nil, fmt.Errorf("{{.Method}} not implemented")
}

`

	methodTmpl, err := template.New("handlerMethod").Parse(handlerMethodTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse handler method template: %w", err)
	}

	for _, rpc := range newHandlerRPCs {
		err = methodTmpl.Execute(&hb, struct {
			Method string
			Prefix string
			Req    string
			Resp   string
		}{
			Method: rpc.Method,
			Prefix: prefix,
			Req:    rpc.Req,
			Resp:   rpc.Resp,
		})
		if err != nil {
			return fmt.Errorf("failed to execute handler method template: %w", err)
		}
	}

	if err := os.WriteFile(handlerPath, hb.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write handler file: %w", err)
	}

	return nil
}

// extractRPCsFromProto parses a .proto file and extracts RPC definitions, excluding Health
func extractRPCsFromProto(file string) ([]string, []struct{ Method, Req, Resp string }, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read proto file %s: %w", file, err)
	}

	content := string(data)
	commentRegex := regexp.MustCompile(`(?m)//.*$|/\*[\s\S]*?\*/`)
	content = commentRegex.ReplaceAllString(content, "")

	// Match all rpc entries inside base_service, multiline support
	rpcBlockRegex := regexp.MustCompile(`(?m)rpc\s+(\w+)\s*\((\w+)\)\s*returns\s*\((\w+)\)\s*(?:\{[^}]*\}|;)?`)
	matches := rpcBlockRegex.FindAllStringSubmatch(content, -1)

	var rpcs []string
	var details []struct {
		Method string
		Req    string
		Resp   string
	}

	for _, match := range matches {
		if len(match) != 4 {
			continue
		}
		method := match[1]
		if method == "Health" {
			continue
		}
		req := match[2]
		resp := match[3]
		signature := fmt.Sprintf("%s(ctx context.Context, req *pb.%s) (*pb.%s, error)", method, req, resp)
		rpcs = append(rpcs, signature)
		details = append(details, struct {
			Method string
			Req    string
			Resp   string
		}{method, req, resp})
	}

	if len(rpcs) == 0 {
		log.Printf("ℹ️ No valid RPCs found in %s", file)
		return nil, nil, nil
	}

	log.Printf("📜 Extracted %d RPCs from %s: %v", len(rpcs), file, rpcs)
	return rpcs, details, nil
}
