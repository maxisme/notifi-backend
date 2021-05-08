package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/maxisme/notifi-backend/crypt"
	"github.com/spf13/cobra"
	"os"
)

var (
	dbConn      *sql.DB
	credentials string
	uuid        string
)

func Handle(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{Use: "credentials",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// connect to db
		var err error
		dbConn, err = sql.Open("postgres", os.Getenv("DB_HOST"))
		Handle(err)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// disconnect from db
		dbConn.Close()
	},
}

func main() {
	//////////
	// edit //
	//////////
	var editCmd = &cobra.Command{
		Use:   "edit",
		Short: "Set custom credentials for user",
		Long:  "Set 25 character custom credentials for a specific user (UUID)",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(credentials) != 25 {
				return fmt.Errorf("credentials must be 25 chars")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			res, err := dbConn.Exec(`
			UPDATE users
			SET credentials = ?
			WHERE UUID=?
			`, crypt.Hash(credentials), uuid)
			Handle(err)

			rowsEffected, err := res.RowsAffected()
			Handle(err)
			if rowsEffected == 0 {
				Handle(errors.New("credentials were not updated"))
			}
		},
	}
	editCmd.Flags().StringVarP(&credentials, "credentials", "c", "", "25 character string")
	Handle(editCmd.MarkFlagRequired("credentials"))
	editCmd.Flags().StringVarP(&uuid, "UUID", "u", "", "UUID of user (hashed)")
	Handle(editCmd.MarkFlagRequired("UUID"))
	rootCmd.AddCommand(editCmd)

	////////////
	// delete //
	////////////
	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Remove credentials of user",
		Long:  "Remove credentials and credential key of user",
		Run: func(cmd *cobra.Command, args []string) {
			res, err := dbConn.Exec(`
			UPDATE users
			SET credentials = NULL, credential_key = NULL
			WHERE UUID=?
			`, uuid)
			Handle(err)

			rowsEffected, err := res.RowsAffected()
			Handle(err)
			if rowsEffected == 0 {
				Handle(errors.New("credentials were not deleted"))
			}
		},
	}
	deleteCmd.Flags().StringVarP(&uuid, "UUID", "u", "", "UUID of user (hashed)")
	Handle(deleteCmd.MarkFlagRequired("UUID"))
	rootCmd.AddCommand(deleteCmd)

	// execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
