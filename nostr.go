package main

import (
	"context"
	"math/rand"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

var (
	ridx      = 0
	allRelays = []string{
		"wss://nostr-pub.wellorder.net",
		"wss://nostr-relay.freeberty.net",
		"wss://nostr.bitcoiner.social",
		"wss://nostr-relay.wlvs.space",
		"wss://nostr.onsats.org",
		"wss://nostr-relay.untethr.me",
		"wss://nostr.semisol.dev",
		"wss://nostr-pub.semisol.dev",
		"wss://nostr-verified.wellorder.net",
		"wss://nostr.drss.io",
		"wss://relay.damus.io",
		"wss://nostr.openchain.fr",
		"wss://nostr.delo.software",
		"wss://relay.nostr.info",
		"wss://relay.minds.com/nostr/v1/ws",
		"wss://nostr.zaprite.io",
		"wss://nostr.oxtr.dev",
		"wss://nostr.ono.re",
		"wss://relay.grunch.dev",
		"wss://relay.cynsar.foundation",
		"wss://nostr.sandwich.farm",
	}
	n = len(allRelays)
)

var _ = func() error {
	rand.Seed(time.Now().Unix())
	rand.Shuffle(n, func(i, j int) {
		allRelays[i], allRelays[j] = allRelays[j], allRelays[i]
	})
	return nil
}()

func querySync(filter nostr.Filter, max int) []nostr.Event {
	ctx := context.Background()
	events := make([]nostr.Event, 0, max)

	for i := 0; i < 4; i++ {
		ridx = ridx + 1

		url := allRelays[ridx%n]
		subctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if r, err := nostr.RelayConnect(subctx, url); err == nil {
			for _, newEvent := range r.QuerySync(subctx, filter) {
				exists := false
				for _, existing := range events {
					if existing.ID == newEvent.ID {
						exists = true
						break
					}
				}
				if !exists {
					events = append(events, newEvent)
					if len(events) >= max {
						return events
					}
				}
			}
			r.Close()
		}
	}

	return events
}
