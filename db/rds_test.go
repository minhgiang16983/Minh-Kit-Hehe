package db

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
)

func TestRdsMysqlDriver(t *testing.T) {
	os.Setenv("AWS_PROFILE", "hsk-app-prod")
	drv := NewDriver(&RdsMysqlDriver{
		Host:         "hsk-aws-app-prod-mariadb.cwb8w6oqa309.us-east-1.rds.amazonaws.com",
		Port:         3306,
		Username:     "backend_read",
		Password:     "",
		DatabaseName: "hasaki_deal",
		AuthMethod:   AuthMethodAWSIAM,
		AwsRegion:    "us-east-1",
	})

	sql.Register("rds", drv)

	db, err := sql.Open("rds", "backend_read:backend_read_password@tcp(hsk-aws-app-prod-mariadb.cwb8w6oqa309.us-east-1.rds.amazonaws.com:3306)/ha_kit?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM deals")
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		total++
	}

	fmt.Println(total)
}
func TestDatabase(t *testing.T) {
	os.Setenv("AWS_PROFILE", "hsk-app-prod")
	db, err := New(&DBConfig{
		Host:         "hsk-aws-app-prod-mariadb.cwb8w6oqa309.us-east-1.rds.amazonaws.com",
		Port:         3306,
		Username:     "backend_read",
		Password:     "",
		DatabaseName: "hasaki_deal",
		AuthMethod:   AuthMethodAWSIAM,
		AwsRegion:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	rows, err := db.Raw("SELECT * FROM deals").Rows()
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		total++
	}

	fmt.Println("total rows", total)
}
