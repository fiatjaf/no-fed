package main

import (
	"encoding/json"

	"github.com/genjidb/genji"
	"github.com/nbd-wtf/go-nostr"
)

var gj, _ = genji.Open(":memory:")

var _ = gj.Exec(`
CREATE TABLE events (
  id text PRIMARY KEY,
  pubkey text,
  kind int,
  ...
)
CREATE INDEX ON events (pubkey, kind)
`)

func cacheEvent(evt nostr.Event) error {
	return gj.Exec(`INSERT INTO events VALUES ?`, evt)
}

func getCachedEvent(id string) *nostr.Event {
	doc, err := gj.QueryDocument(`SELECT * FROM events WHERE id = ?`, id)
	if err != nil {
		return nil
	}
	j, _ := doc.MarshalJSON()
	var evt nostr.Event
	json.Unmarshal(j, &evt)
	return &evt
}

func getReplaceableEvent(pubkey string, kind int) *nostr.Event {
	doc, err := gj.QueryDocument(`SELECT * FROM events WHERE pubkey = ? AND kind = ?`, pubkey, kind)
	if err != nil {
		return nil
	}
	j, _ := doc.MarshalJSON()
	var evt nostr.Event
	json.Unmarshal(j, &evt)
	return &evt
}
