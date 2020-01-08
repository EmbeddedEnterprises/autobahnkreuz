package auth

import (
	"github.com/gammazero/nexus/v3/wamp"
)

// AnonymousAuth is a authenticator which provides a configurable authrole
// for previously unauthenticated clients.
type AnonymousAuth struct {
	AuthRole string
}

// Authenticate assigns an authrole and an authid to the given session.
func (a AnonymousAuth) Authenticate(_ wamp.ID, _ wamp.Dict, _ wamp.Peer) (*wamp.Welcome, error) {
	return &wamp.Welcome{
		Details: wamp.Dict{
			"authid": string(wamp.GlobalID()),
			"authrole": wamp.List{
				a.AuthRole,
			},
			"authprovider": "static",
			"authmethod":   a.AuthMethod(),
		},
	}, nil
}

// AuthMethod returns a string representing the type of the authenticator
// Use the crossbar.io "anonymous" authmethod name here.
func (a AnonymousAuth) AuthMethod() string {
	return "anonymous"
}
