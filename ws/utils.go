package ws

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/go-redis/redis/v7"
)

func StoreMessage(RDB *redis.Client, key1, key2 string, val []byte) error {
	err := RDB.Set(key2, val, WSMessageExpiration).Err()
	if err != nil {
		return err
	}

	err = RDB.LPush(key1, key2).Err()
	if err != nil {
		return err
	}
	return nil
}

func deleteMessage(RDB *redis.Client, key1, key2 string) error {
	err := RDB.Del(key2).Err()
	if err != nil {
		return err // TODO handle when already deleted
	}

	err = RDB.LRem(key1, 1, key2).Err()
	if err != nil {
		return err
	}
	return nil
}

func GetStoredMessages(RDB *redis.Client, key1 string) (vals []string, err error) {
	tx := RDB.TxPipeline()
	key2s := tx.LRange(key1, 0, -1)
	tx.Del(key1)
	_, err = tx.Exec()
	if err != nil {
		return
	}

	for _, key2 := range key2s.Val() {
		v, err := RDB.Get(key2).Result()
		if err == nil {
			vals = append(vals, v)
			RDB.Del(key2)
		}
	}
	return
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func Log(format string, v ...interface{}) {
	str := fmt.Sprintf(format, v...)
	log.Printf("funnels: %s\n", str)
}
