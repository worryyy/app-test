package main

import (
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestMigrationDSNEnablesMultiStatementsAndPreservesParams(t *testing.T) {
	raw := "user:password@tcp(127.0.0.1:3306)/campus?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai"
	value, err := migrationDSN(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := mysqldriver.ParseDSN(value)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.MultiStatements || !cfg.ParseTime || cfg.Loc.String() != "Asia/Shanghai" || cfg.Params["charset"] != "utf8mb4" {
		t.Fatalf("formatted dsn lost required options: %+v", cfg)
	}
}
