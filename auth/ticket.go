package auth

import (
	"context"
	"errors"
	"time"

	"github.com/EmbeddedEnterprises/autobahnkreuz/metrics"
	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	"github.com/deckarep/golang-set"
	superClient "github.com/gammazero/nexus/client"
	"github.com/gammazero/nexus/wamp"
)

// DynamicTicketAuth is an authenticator which performs authentication based on
// a user and its password (i.e. shared secret)
type DynamicTicketAuth struct {
	SharedSecretAuthenticator
	UpstreamAuthFunc string
	AllowResumeToken bool
}

// NewDynamicTicket creates a new DynamicTicketAuth object based on the given parameters
func NewDynamicTicket(authfunc string, authrolefunc string, realm string, invalid mapset.Set, allowtoken bool) (*DynamicTicketAuth, error) {
	x := &DynamicTicketAuth{
		SharedSecretAuthenticator: SharedSecretAuthenticator{
			AuthMethodValue:  "ticket",
			InvalidAuthRoles: invalid,
			Realm:            realm,
			UpstreamGetAuthRolesFunc: authrolefunc,
		},
		UpstreamAuthFunc: authfunc,
		AllowResumeToken: allowtoken,
	}
	return x, nil
}

// Authenticate authenticates requests a ticket (=password) from the user and
// authenticates the user based on its response.
func (a *DynamicTicketAuth) Authenticate(sid wamp.ID, details wamp.Dict, client wamp.Peer) (*wamp.Welcome, error) {
	ctx := context.Background()
	empty := wamp.Dict{}
	authid := wamp.OptionString(details, "authid")
	if authid == "" {
		metrics.IncrementAuth("DynamicTicketAuthenticator", false)
		return nil, errors.New("wamp.error.empty-auth-id")
	}

	// Challenge Extra map is empty since the ticket challenge only asks for a
	// ticket (using authmethod) and provides no additional challenge info.
	err := client.Send(&wamp.Challenge{
		AuthMethod: a.AuthMethod(),
		Extra:      wamp.Dict{},
	})
	if err != nil {
		metrics.IncrementAuth("DynamicTicketAuthenticator", false)
		return nil, err
	}

	// Read AUTHENTICATE response from client.
	// A timeout of 5 seconds should be enough for slow clients...
	msg, err := wamp.RecvTimeout(client, 5*time.Second)
	if err != nil {
		metrics.IncrementAuth("DynamicTicketAuthenticator", false)
		return nil, err
	}
	authRsp, ok := msg.(*wamp.Authenticate)
	if !ok {
		util.Logger.Warningf("Protocol violation from %v: %v", client, msg.MessageType())
		metrics.IncrementAuth("DynamicTicketAuthenticator", false)
		return nil, errors.New(string(wamp.ErrProtocolViolation))
	}

	// We wrap the ticket inside an object to match the signature of crossbar.io here.
	// So the "upstream" authenticator can be used with crossbar.io.
	ticketObj := wamp.Dict{
		"ticket": authRsp.Signature,
	}
	_, err = util.LocalClient.Call(ctx, a.UpstreamAuthFunc, empty, wamp.List{
		a.Realm,
		authid,
		ticketObj,
	}, empty, "")
	if err != nil {
		util.Logger.Warningf("Failed to call `%s`: %v", a.UpstreamAuthFunc, err)

		castErr, ok := err.(superClient.RPCError)

		if !ok {
			metrics.IncrementAuth("DynamicTicketAuthenticator", false)
			return nil, errors.New("wamp.error.internal-error")
		}

		metrics.IncrementAuth("DynamicTicketAuthenticator", false)
		return nil, errors.New(string(castErr.Err.Error))
	}

	welcome, err := a.FetchAndFilterAuthRoles(authid)
	if err != nil {
		metrics.IncrementAuth("DynamicTicketAuthenticator", false)
		return nil, err
	}
	if a.AllowResumeToken && wamp.OptionFlag(authRsp.Extra, "generate-token") {
		resp, err := util.LocalClient.Call(context.Background(), "ee.auth.create-token", nil, wamp.List{
			authid,
		}, nil, "")
		if err == nil {
			x, _ := wamp.AsDict(welcome.Details["authextra"])
			x["resume-token"] = resp.Arguments[0]
			welcome.Details["authextra"] = x
		} else {
			util.Logger.Warningf("Failed to generate token: %v", err)
		}
	}

	metrics.IncrementAuth("DynamicTicketAuthenticator", true)
	return welcome, nil
}
