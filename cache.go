package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

func cacheEvent(evt nostr.Event) error {
	if evt.Kind != 0 && evt.Kind != 1 && evt.Kind != 3 {
		return fmt.Errorf("won't cache event of kind %d", evt.Kind)
	}

	j, _ := json.Marshal(evt)

	// notes
	k := fmt.Sprintf("1:%s", evt.ID)

	// metadata and contact list
	if evt.Kind == 0 || evt.Kind == 3 {
		k = fmt.Sprintf("%d:%s", evt.Kind, evt.PubKey)

		var order time.Time
		pg.Get(&order, "SELECT time FROM cache WHERE key = $1", k)
		if order.After(evt.CreatedAt) {
			// already exists with a newer date
			return nil
		}
	}

	// save event
	_, err := pg.Exec(`
        INSERT INTO cache (key, value, time, expiration)
        VALUES ($1, $2, $3, now() + interval '10 days')
        ON DUPLICATE KEY UPDATE SET expiration = EXCLUDED.expiration
    `, k, j, evt.CreatedAt)

	return err
}

func getCachedNote(id string) *nostr.Event {
	var evt *nostr.Event
	var j string
	pg.Get(&j, "SELECT value FROM cache WHERE id = $1", fmt.Sprintf("1:%s", id))
	json.Unmarshal([]byte(j), evt)
	return evt
}

func getCachedMetadata(pubkey string) *nostr.Event {
	var evt *nostr.Event
	var j string
	pg.Get(&j, "SELECT value FROM cache WHERE id = $1", fmt.Sprintf("0:%s", pubkey))
	json.Unmarshal([]byte(j), evt)
	return evt
}

func getCachedContactList(pubkey string) *nostr.Event {
	var evt *nostr.Event
	var j string
	pg.Get(&j, "SELECT value FROM cache WHERE id = $1", fmt.Sprintf("0:%s", pubkey))
	json.Unmarshal([]byte(j), evt)
	return evt
}

func getNotesForPubkey(pubkey string) []nostr.Event {
	var js []string
	pg.Select(&js, `
        SELECT value FROM cache
        WHERE key LIKE '1:' || $1 || '%'
        ORDER BY time DESC
        LIMIT 100
    `, pubkey)

	evts := make([]nostr.Event, len(js))
	for i, v := range js {
		var evt nostr.Event
		json.Unmarshal([]byte(v), &evt)
		evts[i] = evt
	}
	return evts
}

// TODO: keep track of which user is in each relay
