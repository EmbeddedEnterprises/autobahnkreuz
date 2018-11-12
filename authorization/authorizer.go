package authorization

import (
	"context"
	"errors"

	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/wamp"
)

// DynamicAuthorizer is an authorizer that uses a WAMP RPC call to verify permissions
// for various actions like CALL, SUBSCRIBE, PUBLISH, REGISTER
type DynamicAuthorizer struct {
	PermitDefault      bool
	TrustedAuthRoles   mapset.Set
	UpstreamAuthorizer string
	Realm              string
}

// Authorize checks whether the session `sess` is allowed to send the message `msg`
func (a DynamicAuthorizer) Authorize(sess *wamp.Session, msg wamp.Message) (bool, error) {
	util.Logger.Debug("DynamicAuthorizer: Checking " + msg.MessageType().String())

	roles, _ := extractAuthRoles(sess.Details["authrole"]);

	// We suppress an error at this point, which is not able to occur, due to a check in MultiAuthorizer.
	// If this happens, there is an issue with the code around this, not with the message, so this should panic, please.
	msgType, uri, _ := getMessageURI(msg)
	util.Logger.Debugf("Authorizing %v on %v for roles %v", msgType, uri, roles)

	ctx := context.Background()
	empty := wamp.Dict{}
	session := wamp.Dict{
		"realm":        a.Realm,
		"authprovider": sess.Details["authprovider"],
		"authid":       sess.Details["authid"],
		"session":      sess.ID,
		"authmethod":   sess.Details["authmethod"],
		"authrole":     roles,
	}
	res, err := util.LocalClient.Call(ctx, a.UpstreamAuthorizer, empty, wamp.List{
		session,
		uri,
		msgType,
	}, empty, "")

	util.Logger.Debugf("Finished authorizing %v on %v for roles %v", msgType, uri, roles)

	if err != nil {
		util.Logger.Warningf("Failed to run authorizer: %v", err)
		return a.PermitDefault, err
	}

	if res.Arguments == nil || len(res.Arguments) == 0 {
		util.Logger.Warning("Authorizer returned no result")
		return a.PermitDefault, errors.New("Authorizer returned no result")
	}
	permit, ok := res.Arguments[0].(bool)
	if ok {
		return permit, nil
	}

	det, ok := res.Arguments[0].(map[string]interface{})
	if ok {
		permit, ok = det["allow"].(bool)
		if !ok {
			return permit, nil
		}
	}

	det2, ok := res.Arguments[0].(map[interface{}]interface{})
	if ok {
		permit, ok = det2["allow"].(bool)
		if ok {
			return permit, nil
		}
	}

	util.Logger.Debug("DynamicAuthorizer: Could not determine, if message is allowed or not. Falling back to default.")
	util.Logger.Debugf("%v", a.PermitDefault)

	return a.PermitDefault, nil
}
