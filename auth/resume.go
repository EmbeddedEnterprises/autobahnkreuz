package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/EmbeddedEnterprises/autobahnkreuz/metrics"
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

type token struct {
	AuthID     string
	ExpireDate time.Time
}

// ResumeAuthenticator is an authenticator which performs authentication based
// on a previously created one-time-token.
// It is designed to be used with the normal ticket authenticator.
type ResumeAuthenticator struct {
	SharedSecretAuthenticator
	// Map from token -> Token
	Tokens map[string]token
}

// NewResumeAuthenticator creates a new ResumeAuthenticator based on the given parameters
func NewResumeAuthenticator(authrolefunc string, realm string, invalidRoles mapset.Set) (*ResumeAuthenticator, error) {
	x := &ResumeAuthenticator{
		SharedSecretAuthenticator: SharedSecretAuthenticator{
			AuthMethodValue:  "resume",
			InvalidAuthRoles: invalidRoles,
			Realm:            realm,
			UpstreamGetAuthRolesFunc: authrolefunc,
		},
		Tokens: make(map[string]token),
	}
	return x, nil
}

// Initialize registers the create-new-token-endpoint
func (r *ResumeAuthenticator) Initialize() {
	// Patched to ee to be similar to featureAuthorizer.
	err := util.LocalClient.Register("ee.auth.create-token", r.createNewToken, wamp.Dict{})
	if err != nil {
		util.Logger.Criticalf("Failed to register create-token method!")
		os.Exit(1)
	}
}

func (r *ResumeAuthenticator) createNewToken(_ context.Context, args wamp.List, _, _ wamp.Dict) *client.InvokeResult {
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
	userToken, err := randomHex(64)
	if err != nil {
		return &client.InvokeResult{
			Err: wamp.URI("wamp.error.internal-error"),
		}
	}
	r.Tokens[userToken] = token{
		AuthID:     authid,
		ExpireDate: time.Now().Add(7 * 24 * time.Hour), // one week
	}
	return &client.InvokeResult{
		Args: wamp.List{
			userToken,
		},
	}
}

// Authenticate asks for the users ticket, checks the provided response with the
// list of previously created tokens.
func (r *ResumeAuthenticator) Authenticate(sid wamp.ID, details wamp.Dict, client wamp.Peer) (*wamp.Welcome, error) {
	authid := wamp.OptionString(details, "authid")
	if authid != "resume" {
		metrics.IncrementAuth(r.AuthMethod(), false)
		return nil, errors.New("wamp.error.wrong-auth-id")
	}

	// Challenge Extra map is empty since the ticket challenge only asks for a
	// ticket (using authmethod) and provides no additional challenge info.
	err := client.Send(&wamp.Challenge{
		AuthMethod: r.AuthMethod(),
		Extra:      wamp.Dict{},
	})
	if err != nil {
		metrics.IncrementAuth(r.AuthMethod(), false)
		return nil, err
	}

	// Read AUTHENTICATE response from client.
	// A timeout of 5 seconds should be enough for slow clients...
	msg, err := wamp.RecvTimeout(client, 5*time.Second)
	if err != nil {
		metrics.IncrementAuth(r.AuthMethod(), false)
		return nil, err
	}
	authRsp, ok := msg.(*wamp.Authenticate)
	if !ok {
		util.Logger.Warningf("Protocol violation from %v: %v", client, msg.MessageType())
		metrics.IncrementAuth(r.AuthMethod(), false)
		return nil, errors.New(string(wamp.ErrProtocolViolation))
	}

	token := authRsp.Signature
	tokenObj, ok := r.Tokens[token]
	delete(r.Tokens, token)

	if !ok || time.Now().After(tokenObj.ExpireDate) {
		metrics.IncrementAuth(r.AuthMethod(), false)
		return nil, errors.New("wamp.error.invalid-token")
	}

	authid = tokenObj.AuthID
	util.Logger.Infof("Token login for user %v", authid)
	newTokenRes := r.createNewToken(context.Background(), wamp.List{
		authid,
	}, nil, nil)

	welcome, err := r.FetchAndFilterAuthRoles(authid)
	if err != nil {
		metrics.IncrementAuth(r.AuthMethod(), false)
		return nil, err
	}
	x, _ := wamp.AsDict(welcome.Details["authextra"])
	x["resume-token"] = newTokenRes.Args[0]
	welcome.Details["authextra"] = x

	metrics.IncrementAuth(r.AuthMethod(), true)
	return welcome, nil
}
