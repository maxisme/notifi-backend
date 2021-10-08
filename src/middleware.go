package main

import (
	"fmt"
	"github.com/go-chi/httprate"
	"github.com/guregu/dynamo"
	"os"
	"sync"
	"time"
)

type localCounter struct {
	windowLength time.Duration
	lastEvict    time.Time
	mu           sync.Mutex
	db           *dynamo.DB
}

var bruteForceTable = os.Getenv("BRUTE_FORCE_TABLE_NAME")

type count struct {
	key       uint64    `dynamo:"brute_key,hash"`
	value     int       `dynamo:"value"`
	updatedAt time.Time `dynamo:"updated_dttm"`
}

func getLimitCounter(db *dynamo.DB) httprate.LimitCounter {
	return &localCounter{db: db}
}

var _ httprate.LimitCounter = &localCounter{}

func (c *localCounter) getCounter(key string, window time.Time) *count {
	k := httprate.LimitCounterKey(key, window)
	var cnt count
	if err := c.db.Table(bruteForceTable).Get("brute_key", k).One(&cnt); err != nil || cnt.key == 0 {
		return &count{key: k, value: 0, updatedAt: time.Now()}
	}
	return &cnt
}

func (c *localCounter) Increment(key string, currentWindow time.Time) error {
	if err := c.evict(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	v := c.getCounter(key, currentWindow)
	v.value += 1

	fmt.Printf("%+v\n", v)
	if err := c.db.Table(bruteForceTable).Put(&v).Run(); err != nil {
		return err
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

func (c *localCounter) evict() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	d := c.windowLength * 3

	if time.Since(c.lastEvict) < d {
		return nil
	}

	var counters []count
	if err := c.db.Table(bruteForceTable).Scan().Filter("'updated_dttm' >= ?", time.Now().Add(-d)).All(&counters); err != nil {
		return err
	}

	var keys []dynamo.Keyed
	for _, v := range counters {
		keys = append(keys, dynamo.Keys{v.key, v.updatedAt})
	}
	_, err := c.db.Table(bruteForceTable).Batch("brute_key", "updated_dttm").Write().Delete(keys...).Run()
	return err
}
