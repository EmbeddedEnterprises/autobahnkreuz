package auth

import (
	"errors"

	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/wamp"
)

type AuthRoles []string

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
		return nil, errors.New("Unable to get roles from rolesRawInterface.")
	}

	return &roles, nil

}

func (this AuthRoles) checkTrustedAuthRoles(trustedAuthRoles mapset.Set) bool {
	if trustedAuthRoles.Cardinality() > 0 {
		// Trusted auth roles are an abstract concept used to reduce network
		// load and latency for often-published topic.
		// When adding your system role to the trusted auth roles, it can save up
		// to 80% bandwidth
		for _, role := range this {
			if trustedAuthRoles.Contains(role) {
				return true
			}
		}
	}

	return false
}
