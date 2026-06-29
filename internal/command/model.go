package command

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ModelCmd returns the model command
func ModelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "model [sql-file-path]",
		Short: "Generate models and store layer from SQL schema",
		Long:  `Generate Go models and store layer from a SQL schema file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sqlPath := args[0]
			if err := generateModelAndStoreFromSQL(sqlPath); err != nil {
				log.Fatalf("❌ Failed to generate model and store: %v", err)
			}
			fmt.Println("✅ Model and store generated successfully")
		},
	}
}

type Field struct {
	Name string
	Type string
	Tag  string
}

type Table struct {
	Name   string
	Fields []Field
}

func generateModelAndStoreFromSQL(sqlPath string) error {
	data, err := os.ReadFile(sqlPath)
	if err != nil {
		return fmt.Errorf("cannot read SQL file: %w", err)
	}

	tables, err := parseSQLToModels(string(data))
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	for _, table := range tables {
		modelCode := generateModelCode(table.Name, table.Fields)
		storeCode := generateStoreCode(table.Name)

		// Create directories if they don't exist
		_ = os.MkdirAll("internal/models", 0755)
		_ = os.MkdirAll("internal/stores", 0755)

		modelFile := filepath.Join("internal/models", table.Name+".go")
		storeFile := filepath.Join("internal/stores", table.Name+"_store.go")

		if fileExists(modelFile) {
			fmt.Printf("⚠️  Skip model: %s\n", modelFile)
		} else {
			_ = os.WriteFile(modelFile, []byte(modelCode), 0644)
			fmt.Printf("✅ Created model: %s\n", modelFile)
		}

		if fileExists(storeFile) {
			fmt.Printf("⚠️  Skip store: %s\n", storeFile)
		} else {
			_ = os.WriteFile(storeFile, []byte(storeCode), 0644)
			fmt.Printf("✅ Created store: %s\n", storeFile)
		}
	}
	// Generate main.go file in stores directory
	mainCode := generateMainCode(tables)
	mainFile := filepath.Join("internal/stores", "main.go")
	if fileExists(mainFile) {
		fmt.Printf("⚠️ overwrite  main: %s\n", mainFile)
		_ = os.WriteFile(mainFile, []byte(mainCode), 0644)
		fmt.Printf("✅ overwrite main: %s\n", mainFile)
	} else {
		_ = os.WriteFile(mainFile, []byte(mainCode), 0644)
		fmt.Printf("✅ Created main: %s\n", mainFile)
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func parseSQLToModels(sql string) ([]Table, error) {
	lines := strings.Split(sql, "\n")
	var tables []Table
	var current Table
	var inCreate bool

	for _, line := range lines {
		line = strings.ToLower(strings.TrimSpace(line))

		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "create index") {
			continue
		}

		// Start CREATE TABLE
		if strings.HasPrefix(line, "create table") {
			if inCreate && current.Name != "" && len(current.Fields) > 0 {
				tables = append(tables, current)
			}
			inCreate = true
			current = Table{}

			re := regexp.MustCompile(`create table(?: if not exists)? (\w+)`)
			match := re.FindStringSubmatch(line)
			if len(match) >= 2 {
				current.Name = match[1]
			}
			continue
		}

		// End CREATE TABLE
		if inCreate && strings.HasPrefix(line, ")") {
			if current.Name != "" && len(current.Fields) > 0 {
				tables = append(tables, current)
			}
			inCreate = false
			continue
		}

		// Process column definition lines
		if inCreate {
			line = strings.TrimSuffix(line, ",")
			re := regexp.MustCompile(`^(\w+)\s+([a-zA-Z0-9_()]+)`)
			match := re.FindStringSubmatch(line)
			if len(match) >= 3 {
				columnName := strings.ToLower(match[1])
				sqlType := strings.ToLower(match[2])
				goType := sqlTypeToGo(sqlType)
				tag := fmt.Sprintf("`gorm:\"column:%s\"`", columnName)

				field := Field{
					Name: toPascalCaseModel(columnName),
					Type: goType,
					Tag:  tag,
				}
				current.Fields = append(current.Fields, field)
			}
		}
	}

	return tables, nil
}

func sqlTypeToGo(sqlType string) string {
	sqlType = strings.ToLower(sqlType)
	switch {
	case strings.Contains(sqlType, "bigint") && strings.Contains(sqlType, "unsigned"):
		return "uint64"
	case strings.Contains(sqlType, "int"):
		return "int64"
	case strings.Contains(sqlType, "bool"):
		return "bool"
	case strings.Contains(sqlType, "varchar"), strings.Contains(sqlType, "text"):
		return "string"
	case strings.Contains(sqlType, "datetime"), strings.Contains(sqlType, "timestamp"):
		return "time.Time"
	case strings.Contains(sqlType, "json"):
		return "datatypes.JSON"
	default:
		return "string"
	}
}

func toPascalCaseModel(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = cases.Title(language.English).String(part)
	}

	result := strings.Join(parts, "")
	// Properly capitalize common abbreviations
	replacer := strings.NewReplacer(
		"Id", "ID",
		"Url", "URL",
		"Uri", "URI",
		"Json", "JSON",
	)
	return replacer.Replace(result)
}

func toCamel(s string) string {
	s = toSingular(s) // convert "tokens" → "token"
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = cases.Title(language.English).String(parts[i])
	}
	return strings.Join(parts, "")
}

func generateModelCode(table string, fields []Field) string {
	var b strings.Builder
	b.WriteString("package models\n\n")

	// Collect necessary imports
	imports := make(map[string]bool)
	for _, f := range fields {
		if f.Type == "datatypes.JSON" {
			imports["gorm.io/datatypes"] = true
		}
		if f.Type == "gorm.DeletedAt" {
			imports["gorm.io/gorm"] = true
		}
		if f.Type == "time.Time" {
			imports["time"] = true
		}
	}

	// Write imports
	if len(imports) > 0 {
		b.WriteString("import (\n")
		if imports["time"] {
			b.WriteString("\t\"time\"\n")
		}
		if imports["gorm.io/datatypes"] {
			b.WriteString("\t\"gorm.io/datatypes\"\n")
		}
		if imports["gorm.io/gorm"] {
			b.WriteString("\t\"gorm.io/gorm\"\n")
		}
		b.WriteString(")\n\n")
	}

	// Struct definition
	structName := toPascalCaseModel(toSingular(table))
	b.WriteString(fmt.Sprintf("// %s represents the `%s` table.\n", structName, table))
	b.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	// Align field names and types
	maxNameLen, maxTypeLen := 0, 0
	for _, f := range fields {
		name := toPascalCaseModel(f.Name)
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
		if len(f.Type) > maxTypeLen {
			maxTypeLen = len(f.Type)
		}
	}

	for _, f := range fields {
		fieldName := toPascalCaseModel(f.Name)
		b.WriteString(fmt.Sprintf("\t%-*s  %-*s  %s\n", maxNameLen, fieldName, maxTypeLen, f.Type, f.Tag))
	}
	b.WriteString("}\n\n")

	// TableName method
	b.WriteString(fmt.Sprintf("// TableName sets the actual table name for the %s model.\n", structName))
	b.WriteString(fmt.Sprintf("func (%s) TableName() string {\n\treturn \"%s\"\n}\n", structName, table))

	return b.String()
}

func generateStoreCode(table string) string {
	structName := toCamel(table)
	module := getCurrentFolderName()

	return fmt.Sprintf(`package stores

import (
	"context"
	"github.com/minhgiang16983/%s/internal/models"
	"gorm.io/gorm"
)

type %sStore struct {
	db *gorm.DB
}

func New%sStore(db *gorm.DB) *%sStore {
	return &%sStore{db: db}
}

func (s *%sStore) GetByID(ctx context.Context, id string) (*models.%s, error) {
	var m models.%s
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *%sStore) Create(ctx context.Context, m *models.%s) error {
	return s.db.WithContext(ctx).Create(m).Error
}

func (s *%sStore) DeleteByID(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&models.%s{}).Error
}
`, module,
		structName, structName, structName, structName,
		structName, structName, structName,
		structName, structName,
		structName, structName)
}

func getCurrentFolderName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "your-base_service"
	}
	return filepath.Base(wd)
}

func toSingular(name string) string {
	if strings.HasSuffix(name, "ies") {
		// categories → category
		return name[:len(name)-3] + "y"
	}
	if strings.HasSuffix(name, "ses") || strings.HasSuffix(name, "xes") || strings.HasSuffix(name, "zes") {
		// addresses → address, boxes → box, quizzes → quiz
		return name[:len(name)-2]
	}
	if strings.HasSuffix(name, "s") && len(name) > 1 {
		// users → user, tokens → token
		return name[:len(name)-1]
	}
	return name
}

func generateMainCode(tables []Table) string {
	var b strings.Builder
	b.WriteString("package stores\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"gorm.io/gorm\"\n")
	b.WriteString(")\n\n")

	// Generate StoreService struct
	b.WriteString("// StoreService integrates all stores to manage data\n")
	b.WriteString("type StoreService struct {\n")
	for _, table := range tables {
		structName := toCamel(table.Name)
		b.WriteString(fmt.Sprintf("\t%sStore *%sStore\n", structName, structName))
	}
	b.WriteString("}\n\n")

	// Generate NewStoreService function
	b.WriteString("// NewStoreService initializes StoreService with MariaDB\n")
	b.WriteString("func NewStoreService(mariaDB *gorm.DB) *StoreService {\n")
	b.WriteString("\treturn &StoreService{\n")
	for _, table := range tables {
		structName := toCamel(table.Name)
		b.WriteString(fmt.Sprintf("\t\t%sStore: New%sStore(mariaDB),\n", structName, structName))
	}
	b.WriteString("\t}\n")
	b.WriteString("}\n")

	return b.String()
}
