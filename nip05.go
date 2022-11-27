package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fiatjaf/litepub"
	"github.com/nbd-wtf/go-nostr/nip05"
)

func handleNip05(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	name := qs.Get("name")

	if name == "" {
		http.Error(w, "missing the ?name= querystring value", 400)
		return
	}

	response := nip05.WellKnownResponse{
		Names:  make(nip05.Name2KeyMap),
		Relays: make(nip05.Key2RelaysMap),
	}

	// name is in the form 'fulano_at_mastodon.social'
	spl := strings.Split(name, "_at_")
	if len(spl) == 2 {
		pubName := spl[0]
		pubDomain := spl[1]
		actorUrl := pubName + "@" + pubDomain
		actor, err := litepub.FetchActivityPubURL(actorUrl)
		if err != nil {
			log.Debug().Err(err).Str("actor", actorUrl).Msg("failed to fetch pub url")
		} else {
			// get our generated nostr pubkey
			_, pubkey := nostrKeysForPubActor(actor)

			response.Names[name] = pubkey
			response.Relays[name] = []string{s.RelayURL}
		}
	}

	json.NewEncoder(w).Encode(response)
}
