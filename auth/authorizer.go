package auth

import (
	"context"

	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/wamp"
)

type DynamicAuthorizer struct {
	PermitDefault      bool
	TrustedAuthRoles   mapset.Set
	UpstreamAuthorizer string
	Realm              string
}

func (self DynamicAuthorizer) Authorize(sess *wamp.Session, msg wamp.Message) (bool, error) {

	roles, err := extractAuthRoles(sess.Details["authrole"])

	if err != nil {
		return self.PermitDefault, nil
	}

	isTrustedAuthRole := roles.checkTrustedAuthRoles(self.TrustedAuthRoles)

	if isTrustedAuthRole {
		return true, nil
	}

	msgType := ""
	uri := wamp.URI("")

	switch msg.MessageType() {
	case wamp.CALL:
		msgType = "call"
		uri = msg.(*wamp.Call).Procedure
	case wamp.REGISTER:
		msgType = "register"
		uri = msg.(*wamp.Register).Procedure
	case wamp.SUBSCRIBE:
		msgType = "subscribe"
		uri = msg.(*wamp.Subscribe).Topic
	case wamp.PUBLISH:
		msgType = "publish"
		uri = msg.(*wamp.Publish).Topic
	default:
		return self.PermitDefault, nil
	}

	//util.Logger.Debugf("Authorizing %v on %v for roles %v", msgType, uri, roles)

	ctx := context.Background()
	empty := wamp.Dict{}
	session := wamp.Dict{
		"realm":        self.Realm,
		"authprovider": sess.Details["authprovider"],
		"authid":       sess.Details["authid"],
		"session":      sess.ID,
		"authmethod":   sess.Details["authmethod"],
		"authrole":     roles,
	}
	res, err := util.LocalClient.Call(ctx, self.UpstreamAuthorizer, empty, wamp.List{
		session,
		uri,
		msgType,
	}, empty, "")

	if err != nil {
		util.Logger.Warningf("Failed to run authorizer: %v", err)
		return self.PermitDefault, nil
	}

	if res.Arguments == nil || len(res.Arguments) == 0 {
		util.Logger.Warning("Authorizer returned no result")
		return self.PermitDefault, nil
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

	return self.PermitDefault, nil
}
