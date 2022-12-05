package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

func snotekey(id string) string      { return fmt.Sprintf("1:%s", id) }
func anoteskey(pk, id string) string { return fmt.Sprintf("1:%s:%s", pk, id) }
func metadatakey(pk string) string   { return fmt.Sprintf("0:%s", pk) }
func contactskey(pk string) string   { return fmt.Sprintf("3:%s", pk) }

func cacheEvent(evt nostr.Event) {
	if evt.Kind != 0 && evt.Kind != 1 && evt.Kind != 3 {
		log.Warn().Int("kind", evt.Kind).Msg("won't cache event")
		return
	}

	j, _ := json.Marshal(evt)

	// notes
	keys := []string{
		snotekey(evt.ID),
		anoteskey(evt.PubKey, evt.ID),
	}

	// metadata and contact list
	if evt.Kind == 0 || evt.Kind == 3 {
		var k string
		if evt.Kind == 0 {
			k = metadatakey(evt.PubKey)
		} else if evt.Kind == 3 {
			k = contactskey(evt.PubKey)
		}

		var order time.Time
		pg.Get(&order, "SELECT time FROM cache WHERE key = $1", k)
		if order.After(evt.CreatedAt) {
			// already exists with a newer date
			return
		}

		keys = []string{k}
	}

	// save event (using multiple keys)
	for _, k := range keys {
		_, err := pg.Exec(`
            INSERT INTO cache (key, value, time, expiration)
            VALUES ($1, $2, $3, now() + interval '10 days')
            ON CONFLICT (key) DO UPDATE SET expiration = EXCLUDED.expiration
        `, k, j, evt.CreatedAt)
		if err != nil {
			log.Warn().Err(err).Interface("evt", evt).Msg("error caching")
		}
	}
}

func getCachedNote(id string) *nostr.Event {
	var evt *nostr.Event
	var j string
	err := pg.Get(&j, "SELECT value FROM cache WHERE key = $1", snotekey(id))
	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Str("id", id).Msg("error getting cached note")
	}
	json.Unmarshal([]byte(j), evt)
	return evt
}

func getCachedMetadata(pubkey string) *nostr.Event {
	var evt *nostr.Event
	var j string
	err := pg.Get(&j, "SELECT value FROM cache WHERE key = $1", metadatakey(pubkey))
	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Str("pubkey", pubkey).Msg("error getting cached metadata")
	}
	json.Unmarshal([]byte(j), evt)
	return evt
}

func getCachedContactList(pubkey string) *nostr.Event {
	var evt *nostr.Event
	var j string
	err := pg.Get(&j, "SELECT value FROM cache WHERE key = $1", contactskey(pubkey))
	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Str("pubkey", pubkey).Msg("error getting cached contacts")
	}
	json.Unmarshal([]byte(j), evt)
	return evt
}

func getNotesForPubkey(pubkey string) []nostr.Event {
	var js []string
	err := pg.Select(&js, `
        SELECT value FROM cache
        WHERE key LIKE '1:' || $1 || '%'
        ORDER BY time DESC
        LIMIT 100
    `, pubkey)
	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Str("pubkey", pubkey).Msg("error getting cached notes")
	}

	evts := make([]nostr.Event, len(js))
	for i, v := range js {
		var evt nostr.Event
		json.Unmarshal([]byte(v), &evt)
		evts[i] = evt
	}
	return evts
}

// TODO: keep track of which user is in each relay
