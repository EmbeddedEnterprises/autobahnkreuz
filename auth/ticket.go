package auth

import (
	"context"
	"errors"
	"time"

	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/wamp"
)

type DynamicTicketAuth struct {
	SharedSecretAuthenticator
	UpstreamAuthFunc string
	AllowResumeToken bool
}

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

func (self *DynamicTicketAuth) Authenticate(sid wamp.ID, details wamp.Dict, client wamp.Peer) (*wamp.Welcome, error) {
	ctx := context.Background()
	empty := wamp.Dict{}
	authid := wamp.OptionString(details, "authid")
	if authid == "" {
		return nil, errors.New("Unauthorized")
	}

	// Challenge Extra map is empty since the ticket challenge only asks for a
	// ticket (using authmethod) and provides no additional challenge info.
	err := client.Send(&wamp.Challenge{
		AuthMethod: self.AuthMethod(),
		Extra:      wamp.Dict{},
	})
	if err != nil {
		return nil, err
	}

	// Read AUTHENTICATE response from client.
	// A timeout of 5 seconds should be enough for slow clients...
	msg, err := wamp.RecvTimeout(client, 5*time.Second)
	if err != nil {
		return nil, err
	}
	authRsp, ok := msg.(*wamp.Authenticate)
	if !ok {
		util.Logger.Warningf("Protocol violation from %v: %v", client, msg.MessageType())
		return nil, errors.New("Unauthorized")
	}

	// We wrap the ticket inside an object to match the signature of crossbar.io here.
	// So the "upstream" authenticator can be used with crossbar.io.
	ticketObj := wamp.Dict{
		"ticket": authRsp.Signature,
	}
	_, err = util.LocalClient.Call(ctx, self.UpstreamAuthFunc, empty, wamp.List{
		self.Realm,
		authid,
		ticketObj,
	}, empty, "")
	if err != nil {
		util.Logger.Warningf("Failed to call `%s`: %v", self.UpstreamAuthFunc, err)
		return nil, errors.New("Unauthorized")
	}

	welcome, err := self.FetchAndFilterAuthRoles(authid)
	if err != nil {
		return nil, err
	}
	if self.AllowResumeToken && wamp.OptionFlag(authRsp.Extra, "generate-token") {
		resp, err := util.LocalClient.Call(context.Background(), "embent.auth.create-token", nil, wamp.List{
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
	return welcome, nil
}
