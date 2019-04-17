package multiauthorizer

import (
	"github.com/EmbeddedEnterprises/autobahnkreuz/util"
	nexus "github.com/gammazero/nexus/router"
	"github.com/gammazero/nexus/wamp"
)

type ConsentMode int

const (
	ConsentModeAll ConsentMode = iota
	ConsentModeOne
)

type MultiAuthorizer struct {
	consentMode    ConsentMode
	authorizerList []nexus.Authorizer
}

func New(mode ConsentMode) *MultiAuthorizer {

	util.Logger.Info("Created MultiAuthorizer")
	util.Logger.Infof("Consent Mode: %v", mode)

	var authorizerList []nexus.Authorizer

	return &MultiAuthorizer{
		consentMode:    mode,
		authorizerList: authorizerList,
	}
}

func (mAuth *MultiAuthorizer) Add(authName string, authorizer nexus.Authorizer) {

	util.Logger.Info("Adding Authorizer to MultiAuthorizer")
	util.Logger.Infof("Name: %s", authName)
	mAuth.authorizerList = append(mAuth.authorizerList, authorizer)
}

func (mAuth *MultiAuthorizer) Authorize(sess *wamp.Session, msg wamp.Message) (bool, error) {

	lastAuthResult := false

	for _, singleAuthorizer := range mAuth.authorizerList {
		authResult, authErr := singleAuthorizer.Authorize(sess, msg)

		if authErr != nil {
			return false, authErr
		}

		// If one authorizer approves this message, nothing more must be checked and the message gets approved.
		if authResult && mAuth.consentMode == ConsentModeOne {
			return true, nil
		}

		// If one authorizer denies this message, nothing more must be checked and the message gets rejected.
		if !authResult && mAuth.consentMode == ConsentModeAll {
			return false, nil
		}

		// Otherwise we will save our authResult in a variable and iterate over to the next authorizer,
		// if there is one. Otherwise the last value will be returned and the message gets approved or rejected.
		lastAuthResult = authResult
	}

	return lastAuthResult, nil

}
