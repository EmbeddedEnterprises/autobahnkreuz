package auth

import (
	"context"
	"errors"
	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/wamp"
)

type SharedSecretAuthenticator struct {
	Realm                    string
	UpstreamGetAuthRolesFunc string
	InvalidAuthRoles         mapset.Set
	AuthMethodValue          string
}

func (self *SharedSecretAuthenticator) AuthMethod() string {
	return self.AuthMethodValue
}

func (self *SharedSecretAuthenticator) FetchAndFilterAuthRoles(authid string) (*wamp.Welcome, error) {
	ctx := context.Background()
	empty := wamp.Dict{}
	result, err := util.LocalClient.Call(ctx, self.UpstreamGetAuthRolesFunc, empty, wamp.List{
		self.Realm,
		authid,
	}, empty, "")
	if err != nil {
		util.Logger.Warningf("Failed to call `%s`: %v", self.UpstreamGetAuthRolesFunc, err)
		return nil, errors.New("Unauthorized")
	}
	if len(result.Arguments) == 0 {
		util.Logger.Warningf("Upstream auth func returned no values")
		return nil, errors.New("Unauthorized")
	}

	authroles, isList := wamp.AsList(result.Arguments[0])
	// There is a additional way to provide data for the router regarding client.
	userData, isDict := wamp.AsDict(result.Arguments[0])

	if !isList && !isDict {
		util.Logger.Warningf("Upstream auth func returned no authroles")
		return nil, errors.New("Unauthorized")
	}

	if isDict {
		authroles, isList = wamp.AsList(userData["authroles"])
		if !isList {
			util.Logger.Warningf("Upstream auth func returned no authroles in authextra")
			return nil, errors.New("Unauthorized")
		}
	} else {
		userData = make(map[string]interface{})
	}

	var authRoleList []string
	var targetList []string
	for _, x := range authroles {
		if role, ok := wamp.AsString(x); ok {
			authRoleList = append(authRoleList, role)
		}
	}

	if self.InvalidAuthRoles != nil {
		rawAuthRoles := mapset.NewSet()
		for _, x := range authRoleList {
			rawAuthRoles.Add(x)
		}

		filteredSet := rawAuthRoles.Difference(self.InvalidAuthRoles)

		for x := range filteredSet.Iter() {
			role := x.(string)
			targetList = append(targetList, role)
		}

	} else {
		targetList = authRoleList
	}

	return &wamp.Welcome{
		Details: wamp.Dict{
			"authid":       authid,
			"authrole":     targetList,
			"authextra":    userData,
			"authprovider": "dynamic",
			"authmethod":   self.AuthMethodValue,
		},
	}, nil
}
