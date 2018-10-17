package auth

import (
	"github.com/EmbeddedEnterprises/autobahnkreuz/metrics"
	"github.com/gammazero/nexus/wamp"
)

// AnonymousAuth is a authenticator which provides a configurable authrole
// for previously unauthenticated clients.
type AnonymousAuth struct {
	AuthRole string
}

// Authenticate assigns an authrole and an authid to the given session.
func (a AnonymousAuth) Authenticate(_ wamp.ID, _ wamp.Dict, _ wamp.Peer) (*wamp.Welcome, error) {
	// increasing the count of the anonymous role in metrics
	metrics.IncrementAuth(metrics.Anonymous, true)
	// Authenticate
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
