package auth

import (
	"context"
	"errors"

	"github.com/EmbeddedEnterprises/autobahnkreuz/metrics"
	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/wamp"
)

// SharedSecretAuthenticator is a base type of authenticators which operate on
// shared secrets like passwords and tokens.
type SharedSecretAuthenticator struct {
	Realm                    string
	UpstreamGetAuthRolesFunc string
	InvalidAuthRoles         mapset.Set
	AuthMethodValue          string
}

// AuthMethod returns a string representing the type of the authenticator
func (s *SharedSecretAuthenticator) AuthMethod() string {
	return s.AuthMethodValue
}

// FetchAndFilterAuthRoles tries to fetch authroles for a previously authenticated
// client based on its authid using the configured UpstreamGetAuthRolesFunc
func (s *SharedSecretAuthenticator) FetchAndFilterAuthRoles(authid string) (*wamp.Welcome, error) {
	ctx := context.Background()
	empty := wamp.Dict{}
	result, err := util.LocalClient.Call(ctx, s.UpstreamGetAuthRolesFunc, empty, wamp.List{
		s.Realm,
		authid,
	}, empty, "")
	if err != nil {
		metrics.IncrementAtomic(metrics.MetricGlobal.RejectedAuthorization)
		util.Logger.Warningf("Failed to call `%s`: %v", s.UpstreamGetAuthRolesFunc, err)
		return nil, errors.New("Unauthorized")
	}
	if len(result.Arguments) == 0 {
		metrics.IncrementAtomic(metrics.MetricGlobal.RejectedAuthorization)
		util.Logger.Warningf("Upstream auth func returned no values")
		return nil, errors.New("Unauthorized")
	}

	authroles, isList := wamp.AsList(result.Arguments[0])
	// There is a additional way to provide data for the router regarding client.
	userData, isDict := wamp.AsDict(result.Arguments[0])

	if !isList && !isDict {
		metrics.IncrementAtomic(metrics.MetricGlobal.RejectedAuthorization)
		util.Logger.Warningf("Upstream auth func returned no authroles")
		return nil, errors.New("Unauthorized")
	}

	if isDict {
		authroles, isList = wamp.AsList(userData["authroles"])
		if !isList {
			metrics.IncrementAtomic(metrics.MetricGlobal.RejectedAuthorization)
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

	if s.InvalidAuthRoles != nil {
		rawAuthRoles := mapset.NewSet()
		for _, x := range authRoleList {
			rawAuthRoles.Add(x)
		}

		filteredSet := rawAuthRoles.Difference(s.InvalidAuthRoles)

		for x := range filteredSet.Iter() {
			role := x.(string)
			targetList = append(targetList, role)
		}

	} else {
		targetList = authRoleList
	}
	welcomeDetails := wamp.Dict{}
	if len(result.Arguments) > 1 {
		if dict, dictok := wamp.AsDict(result.Arguments[1]); dictok {
			welcomeDetails = dict
		}
	}

	welcomeDetails["authid"] = authid
	welcomeDetails["authrole"] = targetList
	welcomeDetails["authextra"] = userData
	welcomeDetails["authprovider"] = "dynamic"
	welcomeDetails["authmethod"] = s.AuthMethodValue
	return &wamp.Welcome{
		Details: welcomeDetails,
	}, nil
}
