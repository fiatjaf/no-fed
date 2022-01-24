package main

import (
	"encoding/json"
	"errors"

	"github.com/fiatjaf/go-nostr"
	"github.com/fiatjaf/relayer"
)

type Relay struct{}

func (relay Relay) Name() string { return "no-fed" }

func (relay Relay) Init() error {
	filters := relayer.GetListeningFilters()
	for _, filter := range filters {
		log.Print(filter)
	}

	return nil
}

func (relay Relay) SaveEvent(evt *nostr.Event) error {
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

func (relay Relay) QueryEvents(
	filter *nostr.EventFilter,
) (events []nostr.Event, err error) {
	return nil, nil
}
