package metrics

import (
	"github.com/gammazero/nexus/metrics"
)

const (
	SucceededAuthorization = "SuccededAuthorization"
	RejectedAuthorization  = "RejectedAuthorization"
	Succeeded              = "Succeeded"
	Rejected               = "Rejected"
	Authentication         = "Authentication"
	AuthRolesClients       = "AuthRolesClients"

	Anonymous = "anonymous"
)

var MetricGlobal *metrics.MetricMap

// ConditionalIncrement is only used in combination with Succeeded or Rejected authorization so no extra catch there
func ConditionalIncrement(permit bool) {

	if permit {
		MetricGlobal.IncrementAtomicUint64Key(SucceededAuthorization)
	} else {
		MetricGlobal.IncrementAtomicUint64Key(RejectedAuthorization)
	}
}

// IncrementAuth atomic increases the counter of a certain authentication method `name` depended on if the authentication succeeded or not
func IncrementAuth(name string, succeeded bool) {
	mp := MetricGlobal.GetSubMap(Authentication)
	smp := mp.GetSubMap(name)
	if succeeded {
		smp.IncrementAtomicUint64Key(Succeeded)
	} else {
		smp.IncrementAtomicUint64Key(Rejected)
	}
}

// IncrementRoles increments every given key by 1
func IncrementRoles(roles []string) {
	for _, v := range roles {
		MetricGlobal.GetSubMap(AuthRolesClients).IncrementAtomicUint64Key(v)
	}
}
