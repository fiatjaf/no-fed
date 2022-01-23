package main

import (
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
	return nil
}

func (relay Relay) QueryEvents(
	filter *nostr.EventFilter,
) (events []nostr.Event, err error) {
	return nil, nil
}
