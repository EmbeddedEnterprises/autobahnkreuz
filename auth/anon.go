package auth

import (
	"github.com/gammazero/nexus/wamp"
)

type AnonymousAuth struct {
	AuthRole string
}

func (self AnonymousAuth) Authenticate(sid wamp.ID, details wamp.Dict, client wamp.Peer) (*wamp.Welcome, error) {
	return &wamp.Welcome{
		Details: wamp.Dict{
			"authid": string(wamp.GlobalID()),
			"authrole": wamp.List{
				self.AuthRole,
			},
			"authprovider": "static",
			"authmethod":   self.AuthMethod(),
		},
	}, nil
}

// Use the crossbar.io "anonymous" authmethod name here.
func (self AnonymousAuth) AuthMethod() string {
	return "anonymous"
}
