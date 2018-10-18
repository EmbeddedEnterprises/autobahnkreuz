package metrics

import (
	"context"

	"github.com/EmbeddedEnterprises/autobahnkreuz/util"
	"github.com/gammazero/nexus/client"
	m "github.com/gammazero/nexus/metrics"
	"github.com/gammazero/nexus/wamp"
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

var MetricGlobal *m.MetricMap

func RegisterMetrics(c *client.Client) (err error) {
	err = c.Register("ee.metrics", metric, wamp.Dict{})
	return
}

func metric(_ context.Context, _ wamp.List, _, _ wamp.Dict) (res *client.InvokeResult) {
	mp, err := MetricGlobal.MetricMapToGoMap()
	if err != nil {
		util.Logger.Errorf("%v", err)
	}
	res = &client.InvokeResult{Args: wamp.List{mp}}
	return
}

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
