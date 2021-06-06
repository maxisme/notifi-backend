package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

func GetDB() (*dynamo.DB, error) {
	sesh, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return dynamo.New(sesh, &aws.Config{Region: aws.String("us-east-1")}), nil
}

func AddItem(db *dynamo.DB, table string, item interface{}) error {
	t := db.Table(table)
	return t.Put(item).Run()
}

func GetItem(db *dynamo.DB, table, keyName, key string) (interface{}, error) {
	t := db.Table(table)

	var i interface{}
	err := t.Get(keyName, key).One(i)
	return i, err
}

func GetItems(db *dynamo.DB, table, keyName, key string) (interface{}, error) {
	t := db.Table(table)

	var i interface{}
	err := t.Get(keyName, key).All(i)
	return i, err
}

func UpdateItem(db *dynamo.DB, table, key string, item interface{}) error {
	t := db.Table(table)
	return t.Update(key, item).Run()
}

func DeleteItem(db *dynamo.DB, table, keyName, key string) error {
	t := db.Table(table)
	return t.Delete(keyName, key).Run()
}
