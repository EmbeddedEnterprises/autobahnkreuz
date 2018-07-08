package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/client"
	"github.com/gammazero/nexus/wamp"
)

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type Token struct {
	AuthID     string
	ExpireDate time.Time
}

type ResumeAuthenticator struct {
	SharedSecretAuthenticator
	// Map from token -> Token
	Tokens map[string]Token
}

func NewResumeAuthenticator(authrolefunc string, realm string, invalidRoles mapset.Set) (*ResumeAuthenticator, error) {
	x := &ResumeAuthenticator{
		SharedSecretAuthenticator: SharedSecretAuthenticator{
			AuthMethodValue:  "resume",
			InvalidAuthRoles: invalidRoles,
			Realm:            realm,
			UpstreamGetAuthRolesFunc: authrolefunc,
		},
		Tokens: make(map[string]Token),
	}
	return x, nil
}

func (self *ResumeAuthenticator) Initialize() {
	err := util.LocalClient.Register("wamp.auth.create-token", self.CreateNewToken, wamp.Dict{})
	if err != nil {
		util.Logger.Criticalf("Failed to register create-token method!")
		os.Exit(1)
	}
}

func (self *ResumeAuthenticator) CreateNewToken(_ context.Context, args wamp.List, _, _ wamp.Dict) *client.InvokeResult {
	if len(args) == 0 {
		return &client.InvokeResult{
			Err: wamp.ErrInvalidArgument,
		}
	}
	authid, ok := wamp.AsString(args[0])
	if !ok || authid == "" {
		return &client.InvokeResult{
			Err: wamp.ErrInvalidArgument,
		}
	}
	token, err := randomHex(64)
	if err != nil {
		return &client.InvokeResult{
			Err: wamp.URI("wamp.error.internal-error"),
		}
	}
	self.Tokens[token] = Token{
		AuthID:     authid,
		ExpireDate: time.Now().Add(7 * 24 * time.Hour), // one week
	}
	return &client.InvokeResult{
		Args: wamp.List{
			token,
		},
	}
}

func (self *ResumeAuthenticator) Authenticate(sid wamp.ID, details wamp.Dict, client wamp.Peer) (*wamp.Welcome, error) {
	authid := wamp.OptionString(details, "authid")
	if authid != "resume" {
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

	token := authRsp.Signature
	tokenObj, ok := self.Tokens[token]
	delete(self.Tokens, token)

	if !ok || time.Now().After(tokenObj.ExpireDate) {
		return nil, errors.New("Unauthorized")
	}

	authid = tokenObj.AuthID
	util.Logger.Infof("Token login for user %v", authid)
	newTokenRes := self.CreateNewToken(context.Background(), wamp.List{
		authid,
	}, nil, nil)

	welcome, err := self.FetchAndFilterAuthRoles(authid)
	if err != nil {
		return nil, err
	}
	x, _ := wamp.AsDict(welcome.Details["authextra"])
	x["resume-token"] = newTokenRes.Args[0]
	welcome.Details["authextra"] = x
	return welcome, nil
}
