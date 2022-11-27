package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/fiatjaf/litepub"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip10"
)

func nostrKeysForPubActor(author string) (string, string) {
	// create a fake nostr keypair for this author using the server private key as the hmac key
	sk := hmac.New(sha256.New, s.PrivateKey.D.Bytes()).Sum([]byte(author))
	privkey := hex.EncodeToString(sk)
	pubkey, _ := nostr.GetPublicKey(privkey)
	go pg.Exec(`
        INSERT INTO keys (pub_actor_url, nostr_privkey, nostr_pubkey)
        VALUES ($1, $2, $3)
        ON CONFLICT DO NOTHING
    `, author, privkey, pubkey)

	return privkey, pubkey
}

func nostrEventFromPubNote(note *litepub.Note) nostr.Event {
	privkey, pubkey := nostrKeysForPubActor(note.AttributedTo)

	tags := make(nostr.Tags, 0, 2)
	// "e" tags
	if note.InReplyTo != "" {
		var id string
		if err := pg.Get(&id, "SELECT nostr_event_id FROM notes WHERE pub_note_url = $1", note.InReplyTo); err == nil {
			tags = append(tags, nostr.Tag{"e", id, s.RelayURL})
		} else {
			if note, err := litepub.FetchNote(note.InReplyTo); err == nil {
				evt := nostrEventFromPubNote(note) // @warn will recurse until the start of the thread
				tags = append(tags, nostr.Tag{"e", evt.ID, s.RelayURL})
			}
		}
	}

	// "p" tags
	for _, a := range append(note.CC, note.To...) {
		if strings.HasSuffix(a, "/followers") || strings.HasSuffix(a, "https://www.w3.org/ns/activitystreams#Public") {
			continue
		}

		_, pk := nostrKeysForPubActor(a)
		tags = append(tags, nostr.Tag{"p", pk, s.RelayURL})
	}

	evt := nostr.Event{
		CreatedAt: note.Published,
		PubKey:    pubkey,
		Tags:      tags,
		Kind:      1,
		Content:   strip.StripTags(note.Content),
	}

	if err := evt.Sign(privkey); err != nil {
		log.Warn().Err(err).Interface("evt", evt).Msg("fail to sign an event")
	}

	go pg.Exec(`
        INSERT INTO notes (pub_note_url, nostr_event_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `, note.Id, evt.ID)

	return evt
}

func nostrEventFromActorMetadata(actor *litepub.Actor) nostr.Event {
	privkey, pubkey := nostrKeysForPubActor(actor.Id)

	name := actor.Name
	if name == "" {
		name = actor.PreferredUsername
	}

	nip05 := ""
	if parsed, err := url.Parse(actor.Id); err == nil {
		domain := parsed.Hostname()
		nip05 = actor.PreferredUsername + "@" + domain
	}

	jmetadata, _ := json.Marshal(nostr.ProfileMetadata{
		Name:    name,
		About:   actor.Summary,
		Picture: actor.Icon.URL,
		NIP05:   nip05,
	})

	evt := nostr.Event{
		CreatedAt: actor.Published,
		PubKey:    pubkey,
		Tags:      make(nostr.Tags, 0),
		Kind:      0,
		Content:   string(jmetadata),
	}

	if err := evt.Sign(privkey); err != nil {
		log.Warn().Err(err).Interface("evt", evt).Msg("fail to sign an event")
	}

	return evt
}

func nostrEventFromActorFollows(actor *litepub.Actor) nostr.Event {
	privkey, pubkey := nostrKeysForPubActor(actor.Id)

	follows, _ := litepub.FetchFollowing(actor.Following)
	tags := make(nostr.Tags, len(follows))
	for i, followedUrl := range follows {
		followedPrivkey, followedPubkey := nostrKeysForPubActor(followedUrl)

		go pg.Exec(`
            INSERT INTO keys (pub_actor_url, nostr_privkey, nostr_pubkey)
            VALUES ($1, $2, $3)
            ON CONFLICT DO NOTHING
        `, followedUrl, followedPrivkey, followedPubkey)

		tags[i] = nostr.Tag{"p", followedPubkey, s.RelayURL}
	}

	evt := nostr.Event{
		CreatedAt: actor.Published,
		PubKey:    pubkey,
		Tags:      tags,
		Kind:      3,
	}

	if err := evt.Sign(privkey); err != nil {
		log.Warn().Err(err).Interface("evt", evt).Msg("fail to sign an event")
	}

	return evt
}

func pubNoteFromNostrEvent(event nostr.Event) litepub.Note {
	pTags := event.Tags.GetAll([]string{"p", ""})
	cc := make([]string, len(pTags))
	for i, tag := range pTags {
		cc[i] = s.ServiceURL + "/pub/user/" + tag.Value()
	}

	inReplyTo := ""
	if replyTag := nip10.GetImmediateReply(event.Tags); replyTag != nil {
		inReplyTo = s.ServiceURL + "/pub/note/" + replyTag.Value()
	}

	return litepub.Note{
		Base: litepub.Base{
			Id:   s.ServiceURL + "/pub/note/" + event.ID,
			Type: "Note",
		},
		Published:    event.CreatedAt,
		AttributedTo: s.ServiceURL + "/pub/user/" + event.PubKey,
		Content:      event.Content,
		InReplyTo:    inReplyTo,
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		CC:           cc,
	}
}

func pubActorFromNostrEvent(event nostr.Event) litepub.Actor {
	metadata, _ := nostr.ParseMetadata(event)

	return litepub.Actor{
		Base: litepub.Base{
			Id:   s.ServiceURL + "/pub/user/" + event.PubKey,
			Type: "Person",
		},
		URL:                       s.ServiceURL + "/" + event.PubKey,
		ManuallyApprovesFollowers: false,
		Published:                 event.CreatedAt,
		Followers:                 s.ServiceURL + "/pub/user/" + event.PubKey + "/followers",
		Following:                 s.ServiceURL + "/pub/user/" + event.PubKey + "/following",
		Inbox:                     s.ServiceURL + "/pub",
		Outbox:                    s.ServiceURL + "/pub/user/" + event.PubKey + "/outbox",
		PreferredUsername:         event.PubKey,
		Name:                      metadata.Name,
		Summary:                   metadata.About,
		Icon: litepub.ActorImage{
			Type: "Image",
			URL:  metadata.Picture,
		},
		PublicKey: litepub.PublicKey{
			Id:           s.ServiceURL + "/pub/user/" + event.PubKey + "#main-key",
			Owner:        s.ServiceURL + "/pub/user/" + event.PubKey,
			PublicKeyPEM: s.PublicKeyPEM,
		},
	}
}

func isPublicKey(pubkey string) bool {
	v, err := hex.DecodeString(pubkey)
	if err != nil {
		return false
	}
	return len(v) == 32
}

func isNoteId(noteId string) bool {
	v, err := hex.DecodeString(noteId)
	if err != nil {
		return false
	}
	return len(v) == 32
}
