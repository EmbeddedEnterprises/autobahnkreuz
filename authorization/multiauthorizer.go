package authorization

import (
	"github.com/EmbeddedEnterprises/autobahnkreuz/util"
	"github.com/gammazero/nexus/router"
	"github.com/gammazero/nexus/wamp"
)

type MultiAuthorizer struct {
	Authorizer []router.Authorizer
}

/* We want to iterate over every authorizer.
 * If an authorizer emits an error, we want to stop iteration
 *
 */
func (mAuth *MultiAuthorizer) AuthorizeEvery(sess *wamp.Session, msg wamp.Message) (bool, error) {
	for _, Authorizer := range mAuth.Authorizer {
		isAuthorized, authError := Authorizer.Authorize(sess, msg);
		if isAuthorized {
			return true, nil
		}

		if authError != nil {
			return false, authError
		}
	}

	return false, nil
}

func (mAuth *MultiAuthorizer) Authorize(sess *wamp.Session, msg wamp.Message) (bool, error) {
	isAuthorized, authError := mAuth.AuthorizeEvery(sess, msg);
	util.Logger.Info(msg, isAuthorized)


	if authError != nil {
		util.Logger.Error(authError)
	}

	return isAuthorized, authError

}