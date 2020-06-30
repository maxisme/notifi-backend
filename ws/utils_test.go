package ws

import (
	"sync"
	"testing"
)

func TestGetStoredMessages(t *testing.T) {
	numRoutines := 100
	numMessages := 10
	messageSize := 1000

	key1 := randString(100)

	wg := sync.WaitGroup{}
	wg.Add(numRoutines)
	for i := 0; i < numRoutines; i++ {
		go func() {
			defer wg.Done()
			var key2 string
			for i := 0; i < numMessages; i++ {
				key2 = randString(100)
				msg := []byte(randString(messageSize))
				err := StoreMessage(RDB, key1, key2, msg)
				if err != nil {
					t.Errorf(err.Error())
					return
				}
			}

			// delete message
			if err := DeleteMessage(RDB, key1, key2); err != nil {
				t.Errorf(err.Error())
				return
			}

			_, err := GetStoredMessages(RDB, key1)
			if err != nil {
				t.Errorf(err.Error())
				return
			}
		}()
	}
	wg.Wait()

	// check if all messages have already been read and deleted successfully
	messages, err := GetStoredMessages(RDB, key1)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if len(messages) != 0 {
		t.Errorf("invalid # of stored messages expected 0")
		return
	}
}
