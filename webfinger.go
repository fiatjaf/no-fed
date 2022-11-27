package main

import (
	"encoding/json"
	"net/http"

	"github.com/fiatjaf/litepub"
)

func webfinger(w http.ResponseWriter, r *http.Request) {
	name, err := litepub.HandleWebfingerRequest(r)
	if err != nil {
		http.Error(w, "broken webfinger query: "+err.Error(), 400)
		return
	}

	log.Debug().Str("name", name).Msg("got webfinger request")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(litepub.WebfingerResponse{
		Subject: r.URL.Query().Get("resource"),
		Links: []litepub.WebfingerLink{
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: s.ServiceURL + "/pub/user/" + name,
			},
		},
	})
}
