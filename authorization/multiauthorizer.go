package authorization

import (
	"github.com/EmbeddedEnterprises/autobahnkreuz/util"
	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/wamp"
)

type MultiAuthorizer struct {
	Authorizer       []MultiAuthorizerEntry
	PermitDefault    bool
	TrustedAuthRoles mapset.Set
}

type AuthorizeStates int

const (
	AUTHORIZED     AuthorizeStates = 1
	UNAUTHORIZED   AuthorizeStates = 2
	NOT_DETERMINED AuthorizeStates = 3
)

type MultiAuthorizerEntry interface {
	Authorize(sess *wamp.Session, msg wamp.Message, authRoles AuthRoles, msgType string, URI wamp.URI) (AuthorizeStates, error)
}

/* We want to iterate over every authorizer.
 * If an authorizer emits an error, we want to stop iteration and use PermitDefault.
 * If an authorizer returns AUTHORIZED or UNAUTHORIZED, we want to stop iteration and use the resulting value.
 * If an authorizer returns NOT_DETERMINED, the next authorizer is asked.
 */
func (mAuth *MultiAuthorizer) AuthorizeEvery(sess *wamp.Session, msg wamp.Message, authRoles *AuthRoles, msgType string, URI wamp.URI) (bool, error) {
	for _, Authorizer := range mAuth.Authorizer {
		authorizedState, authError := Authorizer.Authorize(sess, msg, *authRoles, msgType, URI)
		if authorizedState == AUTHORIZED {
			return true, nil
		} else if authorizedState == UNAUTHORIZED {
			return false, nil
		} else if authError != nil {
			return mAuth.PermitDefault, authError
		}
	}

	return mAuth.PermitDefault, nil
}

func (mAuth *MultiAuthorizer) Authorize(sess *wamp.Session, msg wamp.Message) (bool, error) {

	if msg.MessageType() != wamp.CALL &&
		msg.MessageType() != wamp.REGISTER &&
		msg.MessageType() != wamp.SUBSCRIBE &&
		msg.MessageType() != wamp.PUBLISH {
		return true, nil
	}

	authRoles, _ := extractAuthRoles(sess.Details["auth_role"])
	msgType, URI, _ := getMessageURI(msg)
	isTrustedAuthRole, _, err := isTrustedAuthRole(sess, mAuth.TrustedAuthRoles)

	if err != nil {
		return mAuth.PermitDefault, err
	}

	if isTrustedAuthRole {
		return true, nil
	}

	isAuthorized, authError := mAuth.AuthorizeEvery(sess, msg, authRoles, msgType, URI);
	util.Logger.Info(msg, isAuthorized)

	if authError != nil {
		util.Logger.Error(authError)
	}

	return isAuthorized, authError

}
