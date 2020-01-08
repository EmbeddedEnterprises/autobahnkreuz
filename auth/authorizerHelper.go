package auth

import (
	"errors"

	mapset "github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/v3/wamp"
)

// authRoles is a list of authroles which can be used to check against a set of authroles
type authRoles []string

// extractAuthRoles converts a result list or string to an instance of authroles
// rolesRawInterface may be a string or a list of strings
func extractAuthRoles(rolesRawInterface interface{}) (*authRoles, error) {
	roles := authRoles{}
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

func (r authRoles) checkTrustedAuthRoles(trustedAuthRoles mapset.Set) bool {
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
