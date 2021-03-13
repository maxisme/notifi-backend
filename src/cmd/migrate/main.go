package main

import (
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/maxisme/notifi-backend/conn"
	"github.com/spf13/cobra"
	"os"
)

func runMigration() error {
	fmt.Println(conn.getPgConString())
	m, err := migrate.New(
		"file://migrations/",
		conn.getPgConString())
	if err != nil {
		return err
	}
	return m.Up()
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Run db migration",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runMigration(); err != nil {
				fmt.Println("ERROR: " + err.Error())
			}
		},
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
