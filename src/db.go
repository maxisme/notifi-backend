package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

func GetDB() (*dynamo.DB, error) {
	sesh := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return dynamo.New(sesh, &aws.Config{Region: aws.String(Region)}), nil
}

func AddItem(db *dynamo.DB, table string, item interface{}) error {
	t := db.Table(table)
	return t.Put(item).Run()
}

func UpdateItem(db *dynamo.DB, table, key string, item interface{}) error {
	t := db.Table(table)
	return t.Update(key, item).Run()
}

//func DeleteItem(db *dynamo.DB, table, keyName, key string) error {
//	t := db.Table(table)
//	return t.Delete(keyName, key).Run()
//}
