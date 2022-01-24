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
-- events on the nostr side (created only from pub incoming things)
CREATE FUNCTION tags_to_tagvalues(jsonb) RETURNS text[]
    AS 'SELECT array_agg(t->>1) FROM (SELECT jsonb_array_elements($1) AS t)s;'
    LANGUAGE SQL
    IMMUTABLE
    RETURNS NULL ON NULL INPUT;

CREATE TABLE IF NOT EXISTS event (
  id text NOT NULL,
  pubkey text NOT NULL,
  created_at integer NOT NULL,
  kind integer NOT NULL,
  tags jsonb NOT NULL,
  content text NOT NULL,
  sig text NOT NULL,

  tagvalues text[] GENERATED ALWAYS AS (tags_to_tagvalues(tags)) STORED
);

CREATE UNIQUE INDEX IF NOT EXISTS ididx ON event (id);
CREATE UNIQUE INDEX IF NOT EXISTS pubkeytimeidx ON event (pubkey, created_at);
CREATE INDEX IF NOT EXISTS arbitrarytagvalues ON event USING gin (tagvalues);

-- things that are translated from pub to nostr
CREATE TABLE followers (
  nostr_pubkey text NOT NULL,
  pub_actor_url text NOT NULL,

  UNIQUE(nostr_pubkey, pub_actor_url)
);
CREATE INDEX IF NOT EXISTS pubfollowersidx ON followers (nostr_pubkey);

CREATE TABLE keys (
  pub_actor_url text NOT NULL,
  nostr_privkey text NOT NULL,
  nostr_pubkey text PRIMARY KEY
);

-- things that exist only on the pub side (created from nostr events received)
CREATE TABLE actors (
  pubkey text PRIMARY KEY,
  created_at int NOT NULL,
  name text,
  about text,
  picture text
);

CREATE TABLE notes (
  id text PRIMARY KEY,
  pubkey text NOT NULL,
  created_at int NOT NULL,
  content text NOT NULL
);
CREATE INDEX IF NO EXISTS notepubkeyidx ON notes (pubkey);
    `)
	relayer.Log.Print(err)
	return db, nil
}
