package main

import (
	"github.com/go-chi/httprate"
	"github.com/guregu/dynamo"
	"net"
	"net/http"
	"os"
	"strings"
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

type Count struct {
	Key       uint64    `dynamo:"brute_key,hash"`
	Value     int       `dynamo:"value"`
	UpdatedAt time.Time `dynamo:"updated_dttm"`
}

func getLimitCounter(db *dynamo.DB) httprate.LimitCounter {
	return &localCounter{db: db}
}

var _ httprate.LimitCounter = &localCounter{}

func (c *localCounter) getCounter(key string, window time.Time) *Count {
	k := httprate.LimitCounterKey(key, window)
	var cnt Count
	if err := c.db.Table(bruteForceTable).Get("brute_key", k).One(&cnt); err != nil || cnt.Key == 0 {
		return &Count{Key: k, Value: 0, UpdatedAt: time.Now()}
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
	v.Value += 1

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

	return curr.Value, prev.Value, nil
}

func (c *localCounter) evict() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	d := c.windowLength * 3

	if time.Since(c.lastEvict) < d {
		return nil
	}

	var counters []Count
	if err := c.db.Table(bruteForceTable).Scan().Filter("'updated_dttm' >= ?", time.Now().Add(-d)).All(&counters); err != nil {
		return err
	}

	var keys []dynamo.Keyed
	for _, v := range counters {
		keys = append(keys, dynamo.Keys{v.Key, v.UpdatedAt})
	}
	_, err := c.db.Table(bruteForceTable).Batch("brute_key", "updated_dttm").Write().Delete(keys...).Run()
	return err
}

func KeyByIP(r *http.Request) (string, error) {
	var ip string

	if cfrip := r.Header.Get("CF-Connecting-IP"); cfrip != "" {
		ip = cfrip
	} else if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		ip = xrip
	} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		i := strings.Index(xff, ", ")
		if i == -1 {
			i = len(xff)
		}
		ip = xff[:i]
	} else {
		var err error
		ip, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
	}

	return ip, nil
}
