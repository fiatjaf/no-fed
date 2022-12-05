package main

import (
	"github.com/fiatjaf/relayer"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func initDB(dburl string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dburl)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
-- reverse key map of pub profiles
CREATE TABLE IF NOT EXISTS keys (
  pub_actor_url text NOT NULL,
  nostr_privkey text NOT NULL,
  nostr_pubkey text PRIMARY KEY
);

-- pub profiles that are following nostr pubkeys
CREATE TABLE IF NOT EXISTS followers (
  nostr_pubkey text NOT NULL,
  pub_actor_url text NOT NULL,

  UNIQUE(nostr_pubkey, pub_actor_url)
);
CREATE INDEX IF NOT EXISTS pubfollowersidx ON followers (nostr_pubkey);

-- reverse map of nostr event ids to pub notes
CREATE TABLE IF NOT EXISTS notes (
  pub_note_url text NOT NULL,
  nostr_event_id text PRIMARY KEY
);

-- event cache
CREATE TABLE IF NOT EXISTS cache (
  key text PRIMARY KEY,
  value text NOT NULL,
  time timestamp,
  expiration timestamp
);
CREATE INDEX IF NOT EXISTS prefixmatch ON cache(key text_pattern_ops);
CREATE INDEX IF NOT EXISTS cachedeventorder ON cache (time);

-- TODO: map of actual nostr pubkeys to relays and of nostr event ids to relays
    `)
	if err != nil {
		relayer.Log.Print(err)
	}
	return db, nil
}
