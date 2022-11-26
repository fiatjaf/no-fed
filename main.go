package main

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"

	"github.com/fiatjaf/litepub"
	"github.com/fiatjaf/relayer"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

type Settings struct {
	ServiceName string `envconfig:"SERVICE_NAME" required:"true"`
	ServiceURL  string `envconfig:"SERVICE_URL" required:"true"`
	Port        string `envconfig:"PORT" required:"true"`
	PostgresURL string `envconfig:"DATABASE_URL" required:"true"`
	IconSVG     string `envconfig:"ICON"`
	Secret      string `envconfig:"SECRET"`

	PrivateKey   *rsa.PrivateKey
	PublicKeyPEM string
}

var (
	s   Settings
	pg  *sqlx.DB
	log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
)

func main() {
	err := envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	// key stuff (needed for the activitypub integration)
	var seed [4]byte
	copy(seed[:], []byte(s.Secret))
	s.PrivateKey, err = litepub.GeneratePrivateKey(seed)
	if err != nil {
		log.Fatal().Err(err).Msg("error deriving private key")
	}
	s.PublicKeyPEM, err = litepub.PublicKeyToPEM(&s.PrivateKey.PublicKey)
	if err != nil {
		log.Fatal().Err(err).Msg("error deriving public key")
	}

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log = log.With().Timestamp().Logger()

	// postgres connection
	pg, err = sqlx.Connect("postgres", s.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't connect to postgres")
	}

	// define routes
	relayer.Router.Path("/icon.svg").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/svg+xml")
			fmt.Fprint(w, s.IconSVG)
			return
		})

	relayer.Router.Path("/pub").Methods("POST").HandlerFunc(pubInbox)
	relayer.Router.Path("/pub/user/{pubkey:[\\d\\w-]+}").Methods("GET").HandlerFunc(pubUserActor)
	relayer.Router.Path("/pub/user/{pubkey:[\\d\\w-]+}/following").Methods("GET").HandlerFunc(pubUserFollowing)
	relayer.Router.Path("/pub/user/{pubkey:[\\d\\w-]+}/followers").Methods("GET").HandlerFunc(pubUserFollowers)
	relayer.Router.Path("/pub/user/{pubkey:[\\d\\w-]+}/outbox").Methods("GET").HandlerFunc(pubOutbox)
	relayer.Router.Path("/pub/note/{id}").Methods("GET").HandlerFunc(pubNote)
	relayer.Router.Path("/.well-known/webfinger").HandlerFunc(webfinger)

	relayer.Router.PathPrefix("/").Methods("GET").Handler(http.FileServer(http.Dir("./static")))

	// start the relay/http server
	relayer.Start(Relay{})
}
