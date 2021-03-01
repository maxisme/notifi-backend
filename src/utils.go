package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
)

// RequiredEnvs verifies envKeys all have values
func RequiredEnvs(envKeys []string) error {
	for _, envKey := range envKeys {
		envValue := os.Getenv(envKey)
		if envValue == "" {
			return fmt.Errorf("missing env variable: '%s'", envKey)
		}
	}
	return nil
}

// UpdateErr returns an error if no rows have been effected
func UpdateErr(res sql.Result, err error) error {
	if err != nil {
		return err
	}

	rowsEffected, err := res.RowsAffected()
	if rowsEffected == 0 {
		return errors.New("no rows effected")
	}
	return err
}
