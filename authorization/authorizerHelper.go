package authorization

import (
	"errors"

	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/wamp"
)

// AuthRoles is a list of authroles which can be used to check against a set of authroles
type AuthRoles []string

// extractAuthRoles converts a result list or string to an instance of authroles
// rolesRawInterface may be a string or a list of strings
func extractAuthRoles(rolesRawInterface interface{}) (*AuthRoles, error) {
	roles := AuthRoles{}
	roleRaw, okStr := rolesRawInterface.(string)
	rolesRaw, okArr := wamp.AsList(rolesRawInterface)
	if okStr {
		roles = append(roles, roleRaw)
	} else if okArr {
		for _, x := range rolesRaw {
			r, ok := x.(string)
			if ok {
				roles = append(roles, r)
			}
		}
	} else {
		return nil, errors.New("Unable to get roles from rolesRawInterface")
	}

	return &roles, nil

}


func isTrustedAuthRole(sess *wamp.Session, trustedAuthRoles mapset.Set) (bool, *AuthRoles, error) {
	roles, err := extractAuthRoles(sess.Details["authrole"])

	if err != nil {
		return false, nil, errors.New("could not extract authrole")
	}

	isTrustedAuthRole := roles.checkTrustedAuthRoles(trustedAuthRoles)

	if isTrustedAuthRole {
		return true, roles, nil
	}

}

func (r AuthRoles) checkTrustedAuthRoles(trustedAuthRoles mapset.Set) bool {
	if trustedAuthRoles.Cardinality() > 0 {
		// Trusted auth roles are an abstract concept used to reduce network
		// load and latency for often-published topic.
		// When adding your system role to the trusted auth roles, it can save up
		// to 80% bandwidth
		for _, role := range r {
			if trustedAuthRoles.Contains(role) {
				return true
			}
		}
	}

	return false
}

func getMessageURI(msg wamp.Message) (string, wamp.URI, error) {
	switch msg.MessageType() {
	case wamp.CALL:
		return "call", msg.(*wamp.Call).Procedure, nil
	case wamp.REGISTER:
		return "register", msg.(*wamp.Register).Procedure, nil
	case wamp.SUBSCRIBE:
		return "subscribe", msg.(*wamp.Subscribe).Topic, nil
	case wamp.PUBLISH:
		return "publish", msg.(*wamp.Publish).Topic, nil
	default:
		// Fixed the same bug as in the Feature Authorizer.
		// TODO: Use references at this point instead of wrong values.
		return "", wamp.URI(""), errors.New("Invalid message type")
	}
}
