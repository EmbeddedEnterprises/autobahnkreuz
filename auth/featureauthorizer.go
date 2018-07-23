package auth

import (
	"context"
	"errors"

	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	"github.com/deckarep/golang-set"

	"github.com/gammazero/nexus/client"
	"github.com/gammazero/nexus/wamp"
)

type FeatureAuthorizer struct {
	PermitDefault    bool
	MatrixURI        string
	MappingURI       string
	TrustedAuthRoles mapset.Set
	FeatureMatrix    *FeatureMatrix
	FeatureMapping   *FeatureMapping
	CallCounter      int
}

type FeatureMatrix map[wamp.URI]map[string]bool
type FeatureMapping map[wamp.URI]wamp.URI

func NewFeatureAuthorizer(permitDefault bool, matrixURI string, mappingURI string, trustedAuthRoles mapset.Set) *FeatureAuthorizer {

	featureAuthorizer := FeatureAuthorizer{}

	util.Logger.Infof("permitDefault: %v", permitDefault)

	featureAuthorizer.PermitDefault = permitDefault
	featureAuthorizer.MatrixURI = matrixURI
	featureAuthorizer.MappingURI = mappingURI
	featureAuthorizer.TrustedAuthRoles = trustedAuthRoles
	featureAuthorizer.FeatureMatrix = nil
	featureAuthorizer.FeatureMapping = nil
	featureAuthorizer.CallCounter = 0

	return &featureAuthorizer
}

func (this *FeatureAuthorizer) Initialize() {

	util.Logger.Infof("Initializing Feature Authroizer..")
	util.Logger.Infof("Registering wamp.featureauth.update")

	// TBD: We can't use wamp.* prefix here, it's restricted to the router-internal meta client. - Martin
	// I just changed it to ee.*, which should be the fitting namespace at this point. - Johann
	err := util.LocalClient.Register("ee.featureauth.update", this.Update, wamp.Dict{})

	if err != nil {
		util.Logger.Warningf("%v", err)
	}
}

func (this *FeatureAuthorizer) Update(_ context.Context, args wamp.List, _, _ wamp.Dict) *client.InvokeResult {

	util.Logger.Infof("Updating Matrix and Mapping.")

	err := this.UpdateMatrix()

	if err != nil {
		return &client.InvokeResult{
			Err: wamp.URI("wamp.error.internal-error"),
		}
	}

	err = this.UpdateMapping()

	if err != nil {
		return &client.InvokeResult{
			Err: wamp.URI("wamp.error.internal-error"),
		}
	}

	return &client.InvokeResult{}
}

func (this *FeatureAuthorizer) UpdateMapping() error {

	ctx := context.Background()
	emptyDict := wamp.Dict{}

	// callArguments is empty right now, but maybe not forever.
	callArguments := wamp.List{}
	callRes, callErr := util.LocalClient.Call(ctx, this.MappingURI, emptyDict, callArguments, emptyDict, "")

	if callErr != nil {
		util.Logger.Warningf("%s was not callable.", this.MappingURI)
		util.Logger.Warningf("%v", callErr)
		return callErr
	}

	util.Logger.Infof("Got callRes: %v", callRes)

	if len(callRes.Arguments) < 1 {
		// First Element cannot be accessed -> Segfault
		util.Logger.Warningf("Invalid Reply from MappingURI")
		return errors.New("Invalid Reply from MappingURI")
	}

	util.Logger.Infof("%v", callRes.Arguments[0])
	mappingRaw, castOkay := callRes.Arguments[0].(map[string]interface{})

	if !castOkay {
		util.Logger.Warningf("Invalid Reply from MappingURI, Cast to map[string][]interface{} was not successful.")
		return errors.New("Invalid Reply from MappingURI")
	}

	newFeatureMapping := make(FeatureMapping)

	for featureItem, endpointURIs := range mappingRaw {
		featureItemURI := wamp.URI(featureItem)

		endpointURIs, castOkay := wamp.AsList(endpointURIs)
		if !castOkay {
			util.Logger.Warningf("Invalid Reply from MappingURI, Cast with wamp.AsList was not successful.")
			return errors.New("Invalid Reply from MappingURI")
		}

		for _, endpointInterface := range endpointURIs {

			endpointString, castOkay := wamp.AsString(endpointInterface)

			if !castOkay {
				util.Logger.Warningf("Invalid Reply from MappingURI, Cast with wamp.AsString was not successful.")
				return errors.New("Invalid Reply from MappingURI")
			}

			endpointURI := wamp.URI(endpointString)
			newFeatureMapping[endpointURI] = featureItemURI
		}
	}

	this.FeatureMapping = &newFeatureMapping
	util.Logger.Infof("Assigned new Feature Mapping: %v", this.FeatureMapping)

	return nil
}

func (this *FeatureAuthorizer) UpdateMatrix() error {

	ctx := context.Background()
	emptyDict := wamp.Dict{}

	// callArguments is empty right now, but maybe not forever.
	callArguments := wamp.List{}
	callRes, callErr := util.LocalClient.Call(ctx, this.MatrixURI, emptyDict, callArguments, emptyDict, "")

	if callErr != nil {
		util.Logger.Warningf("%s was not callable.", this.MatrixURI)
		util.Logger.Warningf("%v", callErr)
		return callErr
	}

	util.Logger.Infof("Got callRes: %v", callRes)

	if len(callRes.Arguments) < 1 {
		// First Element cannot be accessed -> Segfault
		util.Logger.Warningf("Invalid Reply from MatrixURI")
		return errors.New("Invalid Reply from MatrixURI")
	}

	featureMatrixRaw, ok := callRes.Arguments[0].(map[string]interface{})
	newFeatureMatrix := make(FeatureMatrix)

	if !ok {
		util.Logger.Warningf("Invalid Reply from MatrixURI")
		return errors.New("Invalid Reply from MatrixURI")
	}

	for featureItem, authRoleList := range featureMatrixRaw {

		featureItemUri := wamp.URI(featureItem)

		newFeatureMatrix[featureItemUri] = make(map[string]bool)

		authRoleList, ok := wamp.AsDict(authRoleList)

		if !ok {
			util.Logger.Warningf("Invalid Reply from MatrixURI")
			return errors.New("Invalid Reply from MatrixURI")
		}

		for authRole, isAllowed := range authRoleList {
			isAllowed, castOkay := isAllowed.(bool)

			if !castOkay {
				isAllowed = false
			}

			newFeatureMatrix[featureItemUri][authRole] = isAllowed
		}

	}

	// Assign newFeatureMatrix to existing matrix
	this.FeatureMatrix = &newFeatureMatrix
	util.Logger.Infof("Assigned new featureMatrix: %v", this.FeatureMatrix)

	return nil
}

func (this *FeatureAuthorizer) Authorize(sess *wamp.Session, msg wamp.Message) (bool, error) {

	util.Logger.Debugf("Pointer Address from FeatureAuthorizer: %p", &this)
	this.CallCounter++
	util.Logger.Debugf("Call Counter from FeatureAuthorizer: %v", this.CallCounter)

	roles, err := extractAuthRoles(sess.Details["authrole"])

	util.Logger.Debugf("Request: %v Session: %v", msg, sess)

	if err != nil {
		return this.PermitDefault, nil
	}

	util.Logger.Infof("Check for trustedAuthRoles")
	isTrustedAuthRole := roles.checkTrustedAuthRoles(this.TrustedAuthRoles)

	if isTrustedAuthRole {
		util.Logger.Infof("Call was from trusted auth role. Access granted.")
		return true, nil
	}

	if this.FeatureMatrix == nil || this.FeatureMapping == nil {
		util.Logger.Warningf("FeatureMatrix or FeatureMapping is not defined.")
		util.Logger.Warningf("FeatureMapping: %v", this.FeatureMapping)
		util.Logger.Warningf("FeatureMatrix: %v", this.FeatureMatrix)
		return this.PermitDefault, nil
	}

	// Transform endpointURI to featureItem

	var messageURI wamp.URI

	switch msg.MessageType() {
	case wamp.CALL:
		messageURI = msg.(*wamp.Call).Procedure
	case wamp.REGISTER:
		messageURI = msg.(*wamp.Register).Procedure
	case wamp.SUBSCRIBE:
		messageURI = msg.(*wamp.Subscribe).Topic
	case wamp.PUBLISH:
		messageURI = msg.(*wamp.Publish).Topic
	default:
		return this.PermitDefault, nil
	}

	featureMapping := *this.FeatureMapping
	featureURI := featureMapping[messageURI]

	featureMatrix := *this.FeatureMatrix

	for _, authRole := range *roles {
		hasPermission := featureMatrix[featureURI][authRole]

		if hasPermission {
			return true, nil
		}
	}

	return this.PermitDefault, nil
}
