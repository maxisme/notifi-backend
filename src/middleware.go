package main

import (
	"fmt"
	"github.com/go-chi/httprate"
	"github.com/guregu/dynamo"
	"os"
	"sync"
	"time"
)

var bruteForceTable = os.Getenv("BRUTE_FORCE_TABLE_NAME")

func getLimitCounter(db *dynamo.DB) httprate.LimitCounter {
	return &localCounter{db: db}
}

var _ httprate.LimitCounter = &localCounter{}

func (c *localCounter) getCounter(key string, window time.Time) *count {
	k := httprate.LimitCounterKey(key, window)
	var cnt count
	if err := c.db.Table(bruteForceTable).Get("key", k).One(&cnt); err != nil {
		return &count{key: k, value: 0, updatedAt: time.Now()}
	}
	return &cnt
}

func (c *localCounter) Increment(key string, currentWindow time.Time) error {
	c.evict()

	c.mu.Lock()
	defer c.mu.Unlock()

	v := c.getCounter(key, currentWindow)
	v.value += 1
	v.updatedAt = time.Now()

	if err := c.db.Table(bruteForceTable).Put(v).Run(); err != nil {
		fmt.Println(err.Error())
	}

	return nil
}

func (c *localCounter) Get(key string, currentWindow, previousWindow time.Time) (int, int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	curr := c.getCounter(key, currentWindow)
	prev := c.getCounter(key, previousWindow)

	return curr.value, prev.value, nil
}

type localCounter struct {
	windowLength time.Duration
	lastEvict    time.Time
	mu           sync.Mutex
	db           *dynamo.DB
}

type count struct {
	key       uint64    `dynamo:"key,hash"`
	value     int       `dynamo:"value"`
	updatedAt time.Time `dynamo:"updated_dttm"`
}

func (c *localCounter) evict() {
	c.mu.Lock()
	defer c.mu.Unlock()

	d := c.windowLength * 3

	if time.Since(c.lastEvict) < d {
		return
	}

	var counters []count
	if err := c.db.Table(bruteForceTable).Scan().Filter("'updated_dttm' >= ?", time.Now().Add(-d)).All(&counters); err != nil {
		fmt.Println(err.Error())
	}

	tx := c.db.WriteTx()
	for _, v := range counters {
		tx.Delete(c.db.Table(bruteForceTable).Delete("key", v.key))
	}
	if err := tx.Run(); err != nil {
		fmt.Println(err.Error())
	}
}
