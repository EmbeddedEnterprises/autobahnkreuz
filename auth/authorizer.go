package auth

import (
	"context"

	"github.com/EmbeddedEnterprises/autobahnkreuz/metrics"
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

	roles, err := extractAuthRoles(sess.Details["authrole"])

	if err != nil {
		return a.PermitDefault, nil
	}

	isTrustedAuthRole := roles.checkTrustedAuthRoles(a.TrustedAuthRoles)

	if isTrustedAuthRole {
		metrics.MetricGlobal.IncrementAtomicUint64Key(metrics.SucceededAuthorization)
		return true, nil
	}
	metrics.MetricGlobal.IncrementAtomicUint64Key(metrics.RejectedAuthorization)

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
		// Fixed the same bug as in the Feature Authorizer.
		return true, nil
	}

	// util.Logger.Debugf("Authorizing %v on %v for roles %v", msgType, uri, roles)

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

	if err != nil {
		util.Logger.Warningf("Failed to run authorizer: %v", err)
		return a.PermitDefault, nil
	}

	if res.Arguments == nil || len(res.Arguments) == 0 {
		util.Logger.Warning("Authorizer returned no result")
		return a.PermitDefault, nil
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

	return a.PermitDefault, nil
}
