package metrics

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/EmbeddedEnterprises/autobahnkreuz/util"
	"github.com/cornelk/hashmap"
)

// ******************* structs

type metricGeneral struct {
	InMessageCount        *uint64
	OutMessageCount       *uint64
	InTrafficBytesTotal   *uint64
	OutTrafficBytesTotal  *uint64
	Authentication        *hashmap.HashMap
	AuthRolesClients      *hashmap.HashMap
	SuccededAuthorization *uint64
	RejectedAuthorization *uint64
}

type metricAuthentication struct {
	succeded *uint32
	rejected *uint32
}

// MetricGlobal is intended to be used as an quick acccess way to increase and decrease simple values such as `in/outMessageCount` and `..Authorization`
var MetricGlobal = &metricGeneral{}

// Init offers initialization for metric api
func Init(port uint16, expose bool, tls bool) {
	// WARNING: tls not implemented, metrics will be accessible over http
	util.Logger.Debugf("Creating metrics with port: %u", port)
	if expose {
		go startAPI(port)
	}
	MetricGlobal = &metricGeneral{
		InMessageCount:        new(uint64),
		OutMessageCount:       new(uint64),
		InTrafficBytesTotal:   new(uint64),
		AuthRolesClients:      hashmap.New(128),
		Authentication:        hashmap.New(128),
		OutTrafficBytesTotal:  new(uint64),
		SuccededAuthorization: new(uint64),
		RejectedAuthorization: new(uint64),
	}
}

func startAPI(port uint16) {
	http.HandleFunc("/", metricToJSON)
	http.ListenAndServe(":"+strconv.Itoa(int(port)), nil)
}

// metricToJSON creates raw view of current data of MetricGlobal
func metricToJSON(w http.ResponseWriter, r *http.Request) {
	content, err := json.MarshalIndent(MetricGlobal, "", "\t")
	if err != nil {
		util.Logger.Warning("Metrics encounter troubles while marshaling: %v", err)
		return
	}
	util.Logger.Debug("Authorization Roles:" + MetricGlobal.AuthRolesClients.String())
	util.Logger.Debug("Authentications per method:" + MetricGlobal.Authentication.String())
	w.Write(content)
}

func IncrementAtomicMap(hmp *hashmap.HashMap, key string) {
	var amt uint64
	curamt, _ := hmp.GetOrInsert(key, &amt)
	count := (curamt).(*uint64)
	atomic.AddUint64(count, 1)
}

func IncrementAtomic(value *uint64) {
	atomic.AddUint64(value, 1)
}

// ConditionalIncrement is only used in combination with Succeded or Rejected authorization so no extra catch there
func ConditionalIncrement(permit bool) {
	if permit {
		IncrementAtomic(MetricGlobal.SuccededAuthorization)
	} else {
		IncrementAtomic(MetricGlobal.RejectedAuthorization)
	}
}
