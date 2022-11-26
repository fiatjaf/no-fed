package main

import (
	"encoding/json"
	"errors"

	"github.com/fiatjaf/relayer"
	"github.com/nbd-wtf/go-nostr"
)

type Relay struct {
	storage Storage
}

func (r Relay) Name() string {
	return "no-fed"
}

func (r Relay) Storage() relayer.Storage {
	return r.storage
}

func (r Relay) OnInitialized() {}

func (relay Relay) Init() error {
	filters := relayer.GetListeningFilters()
	for _, filter := range filters {
		log.Print(filter)
	}

	return nil
}

func (r Relay) AcceptEvent(evt *nostr.Event) bool {
	// block events that are too large
	jsonb, _ := json.Marshal(evt)
	if len(jsonb) > 10000 {
		return false
	}

	return true
}

type Storage struct{}

func (s Storage) Init() error {
	return nil
}

func (s Storage) SaveEvent(evt *nostr.Event) error {
	// here instead of saving the event we turn it into activitypub things
	if len(evt.Content) > 1000 {
		return errors.New("event content too large")
	}

	switch evt.Kind {
	case nostr.KindSetMetadata:
		var m struct {
			Name    string `json:"name"`
			About   string `json:"about"`
			Picture string `json:"picture"`
		}
		if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
			return errors.New("metadata JSON is invalid")
		}

		_, err := pg.Exec(`
            INSERT INTO actors (pubkey, created_at, name, about, picture)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (pubkey) DO UPDATE SET
                created_at = excluded.created_at,
                name = excluded.name,
                about = excluded.about,
                picture = excluded.picture
              WHERE created_at < excluded.created_at
        `, evt.PubKey, evt.CreatedAt, m.Name, m.About, m.Picture)
		if err != nil {
			log.Error().Err(err).Interface("event", evt).Msg("failed to save set_metadata")
		}
	case nostr.KindTextNote:
		_, err := pg.Exec(`
            INSERT INTO notes (id, pubkey, created_at, content)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (id) DO NOTHING
        `, evt.ID, evt.PubKey, evt.CreatedAt, evt.Content)
		if err != nil {
			log.Error().Err(err).Interface("event", evt).Msg("failed to save text_note")
		}
	case nostr.KindContactList:
	case nostr.KindDeletion:
	default:
		return nil
	}

	return nil
}

func (s Storage) QueryEvents(filter *nostr.Filter) (events []nostr.Event, err error) {
	// TODO search fedi servers
	return nil, nil
}

func (s Storage) DeleteEvent(id string, pubkey string) error {
	// TODO send tombstone to fedi servers?
	return nil
}
