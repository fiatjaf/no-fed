package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/fiatjaf/go-nostr"
	"github.com/fiatjaf/litepub"
	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

type DBNote struct {
	ID        string `db:"id"`
	CreatedAt int64  `db:"created_at"`
	PubKey    string `db:"author"`
	Content   string `db:"content"`
}

func pubUserActor(w http.ResponseWriter, r *http.Request) {
	pubkey := mux.Vars(r)["pubkey"]

	var actor struct {
		Name    string `db:"name"`
		About   string `db:"about"`
		Picture string `db:"picture"`
	}
	err := pg.Get(&actor, `
        SELECT name, about, picture
        FROM actors
        WHERE pubkey = $1
    `, pubkey)
	if err != nil {
		http.Error(w, "User not found", 404)
		return
	}

	image := litepub.ActorImage{
		Type: "Image",
		URL:  actor.Picture,
	}

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(litepub.Actor{
		Base: litepub.Base{
			Context: litepub.CONTEXT,
			Id:      s.ServiceURL + "/pub/user/" + pubkey,
			Type:    "Person",
		},

		Name:                      actor.Name,
		PreferredUsername:         pubkey,
		Followers:                 s.ServiceURL + "/pub/user/" + pubkey + "/followers",
		Following:                 s.ServiceURL + "/pub/user/" + pubkey + "/following",
		ManuallyApprovesFollowers: false,
		Image:                     image,
		Icon:                      image,
		Summary:                   actor.About,
		URL:                       s.ServiceURL + "/" + pubkey,
		Inbox:                     s.ServiceURL + "/pub",
		Outbox:                    s.ServiceURL + "/pub/user/" + pubkey + "/outbox",

		PublicKey: litepub.PublicKey{
			Id:           s.ServiceURL + "/pub/user/" + pubkey + "#main-key",
			Owner:        s.ServiceURL + "/pub/user/" + pubkey,
			PublicKeyPEM: s.PublicKeyPEM,
		},
	})
}

func pubUserFollowers(w http.ResponseWriter, r *http.Request) {
	pubkey := mux.Vars(r)["pubkey"]

	followers := make([]string, 0)

	page := litepub.OrderedCollectionPage{
		Base: litepub.Base{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/user/" + pubkey + "/followers?page=1",
		},
		PartOf:       s.ServiceURL + "/pub/user/" + pubkey + "/followers",
		TotalItems:   len(followers),
		OrderedItems: followers,
	}

	w.Header().Set("Content-Type", "application/activity+json")
	if r.URL.Query().Get("page") != "" {
		page.Base.Context = litepub.CONTEXT
		json.NewEncoder(w).Encode(page)
	} else {
		collection := litepub.OrderedCollection{
			Base: litepub.Base{
				Context: litepub.CONTEXT,
				Type:    "OrderedCollection",
				Id:      s.ServiceURL + "/pub/user/" + pubkey + "/followers",
			},
			First:      page,
			TotalItems: len(followers),
		}
		json.NewEncoder(w).Encode(collection)
	}
}

func pubUserFollowing(w http.ResponseWriter, r *http.Request) {
	pubkey := mux.Vars(r)["pubkey"]

	following := make([]string, 0)

	page := litepub.OrderedCollectionPage{
		Base: litepub.Base{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/user/" + pubkey + "/following?page=1",
		},
		PartOf:       s.ServiceURL + "/pub/user/" + pubkey + "/following",
		TotalItems:   len(following),
		OrderedItems: following,
	}

	w.Header().Set("Content-Type", "application/activity+json")
	if r.URL.Query().Get("page") != "" {
		page.Base.Context = litepub.CONTEXT
		json.NewEncoder(w).Encode(page)
	} else {
		collection := litepub.OrderedCollection{
			Base: litepub.Base{
				Context: litepub.CONTEXT,
				Type:    "OrderedCollection",
				Id:      s.ServiceURL + "/pub/user/" + pubkey + "/following",
			},
			First:      page,
			TotalItems: len(following),
		}
		json.NewEncoder(w).Encode(collection)
	}
}

func pubOutbox(w http.ResponseWriter, r *http.Request) {
	pubkey := mux.Vars(r)["pubkey"]

	var dbnotes []DBNote
	err := pg.Select(&dbnotes, `
        SELECT
            history.id::text AS id,
            pubkey,
            name,
            set_at,
            history.cid
        FROM history
        INNER JOIN head ON history.record_id = head.id
        WHERE pubkey = $1
        ORDER BY history.set_at DESC
    `, pubkey)
	if err == sql.ErrNoRows {
		dbnotes = make([]DBNote, 0)
	} else if err != nil && err != sql.ErrNoRows {
		log.Warn().Err(err).Str("pubkey", pubkey).Msg("error fetching stuff from database")
		http.Error(w, "Failed to fetch activities.", 500)
		return
	}

	notes := make([]litepub.Note, len(dbnotes))
	creates := make([]litepub.Create, len(dbnotes))
	for i, dbnote := range dbnotes {
		notes[i] = makeNote(dbnote)
		creates[i] = litepub.WrapCreate(notes[i], s.ServiceURL+"/pub/create/"+dbnote.ID)
	}

	page := litepub.OrderedCollectionPage{
		Base: litepub.Base{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/user/" + pubkey + "/followers?page=1",
		},
		PartOf:       s.ServiceURL + "/pub/user/" + pubkey + "/followers",
		TotalItems:   len(creates),
		OrderedItems: creates,
	}

	w.Header().Set("Content-Type", "application/activity+json")
	if r.URL.Query().Get("max_id") != "" {
		page.Base.Context = litepub.CONTEXT
		json.NewEncoder(w).Encode(page)
	} else {
		collection := litepub.OrderedCollection{
			Base: litepub.Base{
				Context: litepub.CONTEXT,
				Type:    "OrderedCollection",
				Id:      s.ServiceURL + "/pub/user/" + pubkey + "/outbox",
			},
			First:      page,
			TotalItems: page.TotalItems,
		}
		json.NewEncoder(w).Encode(collection)
	}
}

func pubNote(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	note, err := fetchNote(id)
	if err != nil {
		http.Error(w, "Note not found", 404)
		return
	}

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(note)
}

func fetchNote(id string) (note litepub.Note, err error) {
	var dbnote DBNote
	err = pg.Get(&dbnote, `
        SELECT
            history.id::text AS id,
            pubkey,
            name,
            set_at,
            history.cid
        FROM history
        INNER JOIN head ON history.record_id = head.id
        WHERE history.id = $1
        ORDER BY history.set_at DESC
    `, id)
	if err != nil {
		return
	}

	note = makeNote(dbnote)
	note.Base.Context = litepub.CONTEXT
	return
}

func makeNote(dbnote DBNote) litepub.Note {
	return litepub.Note{
		Base: litepub.Base{
			Id:   s.ServiceURL + "/pub/note/" + dbnote.ID,
			Type: "Note",
		},
		Published:    time.Unix(dbnote.CreatedAt, 0).Format(time.RFC3339),
		AttributedTo: s.ServiceURL + "/pub/user/" + dbnote.PubKey,
		Content:      dbnote.Content,
		To:           "https://www.w3.org/ns/activitystreams#Public",
	}
}

func pubInbox(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)

	j := gjson.ParseBytes(b)
	typ := j.Get("type").String()
	author := j.Get("actor").String()

	// create a fake nostr keypair for this author using the private key as the hmac key
	sk := hmac.New(sha256.New, s.PrivateKey.D.Bytes()).Sum([]byte(author))
	privkey := hex.EncodeToString(sk)
	pubkey, _ := nostr.GetPublicKey(privkey)
	go pg.Exec(`
        INSERT INTO keys (pub_actor_url, nostr_privkey, nostr_pubkey)
        VALUES ($1, $2, $3)
        ON CONFLICT DO NOTHING
    `, author, privkey, pubkey)

	switch typ {
	case "Note":
		content := j.Get("content")
		// save a nostr event here
		log.Print(content)
	case "Follow":
		object := j.Get("object").String()
		parts := strings.Split(object, "/")
		target := parts[len(parts)-1]

		_, err := pg.Exec(`
            INSERT INTO followers (nostr_pubkey, pub_actor_url)
            VALUES ($1, $2)
            ON CONFLICT (nostr_pubkey, pub_actor_url) DO NOTHING
        `, target, author)

		if err != nil && err != sql.ErrNoRows {
			log.Warn().Err(err).Str("actor", author).Str("object", object).
				Msg("error saving Follow")
			http.Error(w, "Failed to accept Follow.", 500)
			return
		}

		actor, err := litepub.FetchActor(author)
		if err != nil || actor.Inbox == "" {
			log.Warn().Err(err).Str("actor", author).
				Msg("didn't found an inbox from the follower")
			http.Error(w, "Wrong Follow request.", 400)
			return
		}

		accept := litepub.Accept{
			Base: litepub.Base{
				Context: litepub.CONTEXT,
				Type:    "Accept",
				Id:      s.ServiceURL + "/pub/accept/" + pubkey,
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
			http.Error(w, "Failed to send Accept.", 503)
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
				http.Error(w, "Failed to accept Undo.", 500)
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
			http.Error(w, "Failed to accept Delete.", 500)
			return
		}
		break
	default:
		log.Info().Str("type", typ).Str("body", string(b)).Msg("got unexpected pub event")
	}

	w.WriteHeader(200)
}

func pubDispatchNote(id, pubkey, content string) {
	create := litepub.WrapCreate(makeNote(DBNote{
		ID:        id,
		PubKey:    pubkey,
		Content:   content,
		CreatedAt: time.Now().Unix(),
	}), s.ServiceURL+"/pub/create/"+id)
	create.Context = litepub.CONTEXT

	var followers []string
	err := pg.Select(&followers, `
        SELECT pub_actor_url FROM followers
        WHERE target = $1
    `, pubkey)
	if err != nil {
		log.Warn().Err(err).Str("pubkey", pubkey).Msg("failed to fetch followers")
		return
	}

	for _, identifier := range followers {
		resp, err := litepub.SendSigned(
			s.PrivateKey,
			s.ServiceURL+"/pub/user/"+pubkey+"#main-key",
			identifier,
			create,
		)
		if err != nil {
			var b []byte
			if resp != nil && resp.Body != nil {
				b, _ = ioutil.ReadAll(resp.Body)
			}
			log.Warn().Err(err).Str("body", string(b)).
				Msg("failed to send Accept")
			continue
		}

		log.Print(resp.Request.Header)
		log.Print(resp.StatusCode)
		var b []byte
		if resp != nil && resp.Body != nil {
			b, _ = ioutil.ReadAll(resp.Body)
			log.Print(string(b))
		}
	}
}
