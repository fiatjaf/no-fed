package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-fed/activity/streams/vocab"
	"github.com/go-fed/apcore"
	"github.com/go-fed/apcore/app"
)

func main() {
	apcore.CmdLineName = func() string { return "no-fed" }

	apcore.Run(NoFed{})
}

type NoFed struct{}

// CALLS MADE AT DATABASE INIT TIME
//
// These calls are made when invoking the "initialize database" command.
// They are orthogonal to all of the other calls during the life of
// program execution.
func (nofed NoFed) CreateTables(c context.Context, db app.Database, apc app.APCoreConfig, debug bool) error {
	return nil
}

// CALLS MADE AT INIT-ADMINISTRATOR TIME
//
// These calls are made when invoking the "initialize administrator"
// command-line command. They are orthogonal to all of the other calls
// during the life of program execution
func (nofed NoFed) OnCreateAdminUser(c context.Context, userID string, d app.Database, apc app.APCoreConfig) error {
	return nil
}

// Start is called at the beginning of a server's lifecycle, after
// configuration processing and after the database connection is opened
// but before web traffic is being served.
//
// If an error is returned, then the startup process fails.
func (nofed NoFed) Start() error {
	return nil
}

// Stop is called at the end of a server's lifecycle, after the web
// servers have stopped serving traffic but before the database is
// closed.
//
// If an error is returned, shutdown continues but an error is reported.
func (nofed NoFed) Stop() error {
	return nil
}

// Returns a pointer to the configuration struct used by the specific
// application. It will be used to save and load from configuration
// files. This object will be passed to SetConfiguration after it is
// loaded from file.
//
// It is expected the Application will return an object with sane
// defaults. The object's struct definition may have struct tags
// supported by gopkg.in/ini.v1 for additional customization. For
// example, the "comment" struct tag is much appreciated by admins.
// Also, it is very important that keys to not collide, so prefix your
// configuration options with a common prefix:
//
//     type MyAppConfig struct {
//         SomeKey string `ini:"my_app_some_key" comment:"Description of this key"`
//     }
//
// This configuration object is intended to be stable for the lifetime
// of a running application. When the command to "serve" is given, this
// function is only called once during application initialization.
//
// The command to "configure" will append these defaults to the guided
// flow. Admins will then be able to inspect the file and modify the
// configuration if desired.
//
// However, sane defaults for an application are incredibly important,
// as the "new" command guides an admin through the creation process
// all the way to serving without interruption. So have sane defaults!
func (nofed NoFed) NewConfiguration() interface{} {
	return nil
}

// Sets the configuration. The parameter's type is the same type that
// is returned by NewConfiguration. Return an error if the configuration
// is invalid.
//
// Provides a read-only interface for some of APCore's config fields.
//
// This configuration object is intended to be stable for the lifetime
// of a running application. When the command to serve, is given, this
// function is only called once during application initialization.
//
// When debug is true, the binary was invoked with the dev flag.
func (nofed NoFed) SetConfiguration(config interface{}, apcc app.APCoreConfig, debug bool) error {
	return nil
}

// The handler for the application's "404 Not Found" webpage.
func (nofed NoFed) NotFoundHandler(_ app.Framework) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not found!")
	})
}

// The handler when a request makes an unsupported HTTP method against
// a URI.
func (nofed NoFed) MethodNotAllowedHandler(_ app.Framework) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "method not allowed!")
	})
}

// The handler for an internal server error.
func (nofed NoFed) InternalServerErrorHandler(_ app.Framework) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "internal server error!")
	})
}

// The handler for a bad request.
func (nofed NoFed) BadRequestHandler(_ app.Framework) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "bad request!")
	})
}

// Web handler for a GET call to the login page.
//
// It should render a login page that POSTs to the "/login" endpoint.
//
// If the URL contains a query parameter "login_error" with a value of
// "true", then it should convey to the user that the email or password
// previously entered was incorrect.
func (nofed NoFed) GetLoginWebHandlerFunc(_ app.Framework) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "no logins allowed!")
	})
}

// Web handler for a GET call to the OAuth2 authorization page.
//
// It should render UX that informs the user that the other application
// is requesting to be authorized as that user to obtain certain scopes.
//
// See the OAuth2 RFC 6749 for more information.
func (nofed NoFed) GetAuthWebHandlerFunc(_ app.Framework) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "no logins allowed!")
	})
}

// Web handler for a call to GET an actor's outbox. The framework
// applies OAuth2 authorizations to fetch a public-only or private
// snapshot of the outbox, and passes it to this handler function.
//
// The builtin ActivityPub handler will use the OAuth authorization.
//
// Returning a nil handler is allowed, and doing so results in only
// ActivityStreams content being served.
func (nofed NoFed) GetOutboxWebHandlerFunc(f app.Framework) func(http.ResponseWriter, *http.Request, vocab.ActivityStreamsOrderedCollectionPage) {
	return func(w http.ResponseWriter, r *http.Request, outbox vocab.ActivityStreamsOrderedCollectionPage) {

	}
}

// Web handler for a call to GET an actor's followers collection. The
// framework has no authorization requirements to view a user's
// followers collection.
//
// Also returns for the corresponding AuthorizeFunc handler, which will
// be applied to both ActivityPub and web requests.
//
// Returning a nil handler is allowed, and doing so results in only
// ActivityStreams content being served. Returning a nil AuthorizeFunc
// results in public access.
func (nofed NoFed) GetFollowersWebHandlerFunc(_ app.Framework) (app.CollectionPageHandlerFunc, app.AuthorizeFunc) {
	return func(w http.ResponseWriter, r *http.Request, followers vocab.ActivityStreamsCollectionPage) {

	}, nil
}

// Web handler for a call to GET an actor's following collection. The
// framework has no authorization requirements to view a user's
// following collection.
//
// Also returns for the corresponding AuthorizeFunc handler, which will
// be applied to both ActivityPub and web requests.
//
// Returning a nil handler is allowed, and doing so results in only
// ActivityStreams content being served. Returning a nil AuthorizeFunc
// results in public access.
func (nofed NoFed) GetFollowingWebHandlerFunc(_ app.Framework) (app.CollectionPageHandlerFunc, app.AuthorizeFunc) {
	return func(w http.ResponseWriter, r *http.Request, following vocab.ActivityStreamsCollectionPage) {

	}, nil
}

// Web handler for a call to GET an actor's liked collection. The
// framework has no authorization requirements to view a user's
// liked collection.
//
// Also returns for the corresponding AuthorizeFunc handler, which will
// be applied to both ActivityPub and web requests.
//
// Returning a nil handler is allowed, and doing so results in only
// ActivityStreams content being served. Returning a nil AuthorizeFunc
// results in public access.
func (nofed NoFed) GetLikedWebHandlerFunc(_ app.Framework) (app.CollectionPageHandlerFunc, app.AuthorizeFunc) {
	return nil, nil
}

// Web handler for a call to GET an actor. The framework has no
// authorization requirements to view a user, like a profile.
//
// Also returns for the corresponding AuthorizeFunc handler, which will
// be applied to both ActivityPub and web requests.
//
// Returning a nil handler is allowed, and doing so results in only
// ActivityStreams content being served. Returning a nil AuthorizeFunc
// results in public access.
func (nofed NoFed) GetUserWebHandlerFunc(_ app.Framework) (app.VocabHandlerFunc, app.AuthorizeFunc) {
	return func(w http.ResponseWriter, r *http.Request, user vocab.Type) {

	}, nil
}

// Builds the HTTP and ActivityPub routes specific for this application.
//
// The database is provided so custom handlers can access application
// data directly, allowing clients to create the custom Fediverse
// behavior their application desires.
//
// The app.Framework provided allows handlers to use common behaviors
// provided by the apcore server framework.
//
// The bulk of typical HTTP application logic is in the handlers created
// by the app.Router. The apcore.app.Router also supports creating routes that
// process and serve ActivityStreams data, but the processing of the
// ActivityPub data itself is handled elsewhere in
// ApplyFederatingCallbacks and/or ApplySocialCallbacks.
func (nofed NoFed) BuildRoutes(r app.Router, db app.Database, f app.Framework) error {
	return nil
}

// app.Paths allows applications to customize endpoints supported by apcore.
//
// See the app.Paths object for the various custom paths and their defaults.
// Returning a zero-value struct is valid.
func (nofed NoFed) Paths() app.Paths {
	return app.Paths{}
}

// StaticServingEnabled indicates whether to enable static serving from
// a local folder. Only takes effect during server startup, dynamically
// changing the return value during runtime has no effect.
func (nofed NoFed) StaticServingEnabled() bool {
	return false
}

// NewIDPath creates a new id IRI path component for the content being
// created.
//
// A peer making a GET request to this path on this server should then
// serve the ActivityPub value provided in this call. For example:
//   "/notes/abcd0123-4567-890a-bcd0-1234567890ab"
//
// Ensure the route returned by NewIDPath will be servable by a handler
// created in the BuildRoutes call.
func (nofed NoFed) NewIDPath(c context.Context, t vocab.Type) (path string, err error) {
	return "", nil
}

// ScopePermitsPrivateGetInbox determines if an OAuth token scope
// permits the bearer to view private (non-Public) messages in an
// actor's inbox.
func (nofed NoFed) ScopePermitsPrivateGetInbox(scope string) (permitted bool, err error) {
	return true, nil
}

// ScopePermitsPrivateGetOutbox determines if an OAuth token scope
// permits the bearer to view private (non-Public) messages in an
// actor's outbox.
func (nofed NoFed) ScopePermitsPrivateGetOutbox(scope string) (permitted bool, err error) {
	return true, nil
}

// DefaultUserPreferences returns an application-specific preferences
// struct to be serialized into JSON and used as initial user app
// preferences.
func (nofed NoFed) DefaultUserPreferences() interface{} {
	return map[string]interface{}{}
}

// DefaultUserPrivileges returns an application-specific privileges
// struct to be serialized into JSON and used as initial user app
// privileges.
func (nofed NoFed) DefaultUserPrivileges() interface{} {
	return map[string]interface{}{}
}

// DefaultAdminPrivileges returns an application-specific privileges
// struct to be serialized into JSON and used as initial user app
// privileges for new admins.
func (nofed NoFed) DefaultAdminPrivileges() interface{} {
	return map[string]interface{}{}
}

// Information about this application's software. This will be shown at
// the command line and used for NodeInfo statistics, as well as for
// user agent information.
func (nofed NoFed) Software() app.Software {
	return app.Software{
		Name:         "no-fed",
		UserAgent:    "no-fed",
		MajorVersion: 1,
		MinorVersion: 0,
		PatchVersion: 0,
		Repository:   "https://github.com/fiatjaf/no-fed",
	}
}
