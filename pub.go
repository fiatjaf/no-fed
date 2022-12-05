package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/fiatjaf/litepub"
	"github.com/gorilla/mux"
	"github.com/nbd-wtf/go-nostr"
	"github.com/tidwall/gjson"
)

func pubUserActor(w http.ResponseWriter, r *http.Request) {
	pubkey := mux.Vars(r)["pubkey"]
	log.Debug().Str("pubkey", pubkey).Msg("got pub actor request")

	// try to get cached set_metadata event
	var evt *nostr.Event
	evt = getCachedMetadata(pubkey)
	if evt == nil {
		// try to get profile information from relays
		events := querySync(nostr.Filter{Authors: []string{pubkey}, Kinds: []int{0}}, 1)
		if len(events) == 0 {
			http.Error(w, "user not found", 404)
			return
		}
		go cacheEvent(events[0])
		evt = &events[0]
	}

	actor := pubActorFromNostrEvent(*evt)

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(actor)
}

func pubUserFollowers(w http.ResponseWriter, r *http.Request) {
	pubkey := mux.Vars(r)["pubkey"]
	log.Debug().Str("pubkey", pubkey).Msg("got followers request")

	var followers []string
	pg.Select(&followers,
		`SELECT pub_actor_url FROM followers WHERE nostr_pubkey = $1`,
		pubkey,
	)

	// TODO: also search for kind-3

	page := litepub.OrderedCollectionPage[string]{
		Base: litepub.Base{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/user/" + pubkey + "/followers?page=1",
		},
		PartOf:       s.ServiceURL + "/pub/user/" + pubkey + "/followers",
		TotalItems:   len(followers),
		OrderedItems: followers,
	}
	jpage, _ := json.Marshal(page)

	w.Header().Set("Content-Type", "application/activity+json")
	if r.URL.Query().Get("page") != "" {
		json.NewEncoder(w).Encode(page)
	} else {
		collection := litepub.OrderedCollection{
			Base: litepub.Base{
				Type: "OrderedCollection",
				Id:   s.ServiceURL + "/pub/user/" + pubkey + "/followers",
			},
			First:      json.RawMessage(jpage),
			TotalItems: len(followers),
		}
		json.NewEncoder(w).Encode(collection)
	}
}

func pubUserFollowing(w http.ResponseWriter, r *http.Request) {
	pubkey := mux.Vars(r)["pubkey"]
	log.Debug().Str("pubkey", pubkey).Msg("got following request")

	var evt *nostr.Event
	evt = getCachedContactList(pubkey)
	if evt == nil {
		// try to get contact list from relays
		events := querySync(nostr.Filter{Authors: []string{pubkey}, Kinds: []int{3}}, 1)
		if len(events) != 0 {
			go cacheEvent(events[0])
			evt = &events[0]
		}
	}

	var following []string
	if evt != nil {
		ptags := evt.Tags.GetAll([]string{"p", ""})
		following = make([]string, len(ptags))
		for i, p := range ptags {
			following[i] = p.Value()
		}
	}

	page := litepub.OrderedCollectionPage[string]{
		Base: litepub.Base{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/user/" + pubkey + "/following?page=1",
		},
		PartOf:       s.ServiceURL + "/pub/user/" + pubkey + "/following",
		TotalItems:   len(following),
		OrderedItems: following,
	}
	jpage, _ := json.Marshal(page)

	w.Header().Set("Content-Type", "application/activity+json")
	if r.URL.Query().Get("page") != "" {
		json.NewEncoder(w).Encode(page)
	} else {
		collection := litepub.OrderedCollection{
			Base: litepub.Base{
				Type: "OrderedCollection",
				Id:   s.ServiceURL + "/pub/user/" + pubkey + "/following",
			},
			First:      json.RawMessage(jpage),
			TotalItems: len(following),
		}
		json.NewEncoder(w).Encode(collection)
	}
}

func pubOutbox(w http.ResponseWriter, r *http.Request) {
	pubkey := mux.Vars(r)["pubkey"]
	log.Debug().Str("pubkey", pubkey).Msg("got outbox request")

	events := getNotesForPubkey(pubkey)

	gatherNotes := func() []nostr.Event {
		evts := querySync(nostr.Filter{Kinds: []int{1}, Authors: []string{pubkey}}, 40)
		for _, evt := range evts {
			go cacheEvent(evt)
		}
		return evts
	}

	if len(events) == 0 {
		events = gatherNotes()
	} else {
		go gatherNotes()
	}

	creates := make([]litepub.Create[litepub.Note], len(events))
	for i, evt := range events {
		note := pubNoteFromNostrEvent(evt)
		creates[i] = litepub.WrapCreate(note, s.ServiceURL+"/pub/create/"+evt.ID)
	}

	page := litepub.OrderedCollectionPage[litepub.Create[litepub.Note]]{
		Base: litepub.Base{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/user/" + pubkey + "/outbox",
		},
		PartOf:       s.ServiceURL + "/pub/user/" + pubkey + "/outbox",
		TotalItems:   len(creates),
		OrderedItems: creates,
	}
	jpage, _ := json.Marshal(page)

	collection := litepub.OrderedCollection{
		Base: litepub.Base{
			Type: "OrderedCollection",
			Id:   s.ServiceURL + "/pub/user/" + pubkey + "/outbox",
		},
		First:      json.RawMessage(jpage),
		TotalItems: page.TotalItems,
	}

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(collection)
}

func pubNote(w http.ResponseWriter, r *http.Request) {
	pubkey := mux.Vars(r)["pubkey"]
	noteId := mux.Vars(r)["id"]

	// it's the same for nostr events
	eventId := noteId

	getNotesForPubkey(pubkey)

	events := querySync(nostr.Filter{IDs: []string{eventId}}, 1)
	if len(events) == 0 {
		http.Error(w, "couldn't find note", 404)
		return
	}
	note := pubNoteFromNostrEvent(events[0])

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(note)
}

func pubInbox(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)

	j := gjson.ParseBytes(b)
	typ := j.Get("type").String()
	actor := j.Get("actor").String()

	_, pubkey := nostrKeysForPubActor(actor)

	switch typ {
	case "Note":
		content := j.Get("content")
		// TODO: save a nostr event here
		log.Print(content)
	case "Follow":
		object := j.Get("object").String()
		parts := strings.Split(object, "/")
		target := parts[len(parts)-1]

		_, err := pg.Exec(`
            INSERT INTO followers (nostr_pubkey, pub_actor_url)
            VALUES ($1, $2)
            ON CONFLICT (nostr_pubkey, pub_actor_url) DO NOTHING
        `, target, actor)

		if err != nil && err != sql.ErrNoRows {
			log.Warn().Err(err).Str("actor", actor).Str("object", object).
				Msg("error saving Follow")
			http.Error(w, "failed to accept Follow", 500)
			return
		}

		actor, err := litepub.FetchActor(actor)
		if err != nil || actor.Inbox == "" {
			log.Warn().Err(err).Str("actor", actor.Id).
				Msg("didn't found an inbox from the follower")
			http.Error(w, "wrong Follow request", 400)
			return
		}

		accept := litepub.Accept{
			Base: litepub.Base{
				Type: "Accept",
				Id:   s.ServiceURL + "/pub/accept/" + pubkey,
			},
			Object: object,
		}
		resp, err := litepub.SendSigned(
			s.PrivateKey,
			s.ServiceURL+"/pub/user/"+pubkey+"#main-key",
			actor.Inbox,
			accept,
		)

		var b []byte
		if resp != nil && resp.Body != nil {
			b, _ = ioutil.ReadAll(resp.Body)
			b, _ = ioutil.ReadAll(resp.Body)
		}
		if err != nil {
			log.Warn().Err(err).Str("body", string(b)).
				Msg("failed to send Accept")
			http.Error(w, "failed to send Accept", 503)
			return
		}
		log.Print(string(b))

		break
	case "Undo":
		switch j.Get("object.type").String() {
		case "Follow":
			actor := j.Get("object.actor").String()
			object := j.Get("object.object").String()
			parts := strings.Split(object, "/")
			pubkey := parts[len(parts)-1]

			_, err := pg.Exec(`
                DELETE FROM followers
                WHERE pub_actor_url = $1 AND nostr_pubkey = $2
            `, actor, pubkey)

			if err != nil && err != sql.ErrNoRows {
				log.Warn().Err(err).Str("actor", actor).Str("object", object).
					Msg("error undoing Follow")
				http.Error(w, "failed to accept Undo", 500)
				return
			}
			break
		}
	case "Delete":
		actor := j.Get("actor").String()

		_, err := pg.Exec(`
            DELETE FROM pub_followers
            WHERE follower = $1
        `, actor)

		if err != nil && err != sql.ErrNoRows {
			log.Warn().Err(err).Str("actor", actor).Msg("error accepting Delete")
			http.Error(w, "failed to accept Delete", 500)
			return
		}
		break
	default:
		log.Info().Str("type", typ).Str("body", string(b)).Msg("got unexpected pub event")
	}

	w.WriteHeader(200)
}
