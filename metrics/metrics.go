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

type MetricGeneral struct {
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
	Succeded *uint64
	Rejected *uint64
}

type displayAuthentication struct {
	Succeded uint64
	Rejected uint64
}

type displayGeneral struct {
	InMessageCount        uint64
	OutMessageCount       uint64
	InTrafficBytesTotal   uint64
	OutTrafficBytesTotal  uint64
	Authentication        map[string]displayAuthentication
	AuthRolesClients      map[string]uint64
	SuccededAuthorization uint64
	RejectedAuthorization uint64
}

// MetricGlobal is intended to be used as an quick acccess way to increase and decrease simple values such as `in/outMessageCount` and `..Authorization`
var MetricGlobal = &MetricGeneral{}

// Init offers initialization for metric api
func Init(port uint16, expose bool, tls bool) {
	// WARNING: tls not implemented, metrics will be accessible over http
	util.Logger.Debugf("Creating metrics with port: %u", port)
	if expose {
		go startAPI(port)
	}
	// Creating these types has almost no impact on startup so this is not dependent on expose
	MetricGlobal = &MetricGeneral{
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
	disMtr, err := processMtr()
	if err != nil {
		util.Logger.Warning("Metrics encounter troubles while converting: %v", err)
		return
	}
	content, err := json.MarshalIndent(disMtr, "", "  ")
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

// IncrementAuth atomic increases the counter of a certain authentication method `name` depended on if the authentication succeded or not
func IncrementAuth(name string, succeded bool) {
	var amt metricAuthentication
	curamt, inserted := MetricGlobal.Authentication.GetOrInsert(name, &amt)
	mtrA := ((curamt).(*metricAuthentication))
	if inserted {
		mtrA.Succeded = new(uint64)
		mtrA.Rejected = new(uint64)
	}
	if succeded {
		atomic.AddUint64(mtrA.Succeded, 1)
	} else {
		atomic.AddUint64(mtrA.Rejected, 1)
	}
}

// IncrementRoles increments every given key by 1
func IncrementRoles(roles []string) {
	for k := range roles {
		var amt uint64
		curamt, _ := MetricGlobal.AuthRolesClients.GetOrInsert(k, &amt)
		val := (curamt).(*uint64)
		atomic.AddUint64(val, 1)
	}
}

func SendHandler() {
	IncrementAtomic(MetricGlobal.OutMessageCount)
}

func RecvHandler() {
	IncrementAtomic(MetricGlobal.InMessageCount)
}

func processMtr() (disMtr displayGeneral, err error) {
	// Setting all single valued fields
	disMtr.InMessageCount = *MetricGlobal.InMessageCount
	disMtr.OutMessageCount = *MetricGlobal.OutMessageCount
	disMtr.RejectedAuthorization = *MetricGlobal.RejectedAuthorization
	disMtr.SuccededAuthorization = *MetricGlobal.SuccededAuthorization
	disMtr.InTrafficBytesTotal = *MetricGlobal.InTrafficBytesTotal
	disMtr.OutTrafficBytesTotal = *MetricGlobal.OutTrafficBytesTotal

	// initialize maps
	disMtr.AuthRolesClients = make(map[string]uint64)
	disMtr.Authentication = make(map[string]displayAuthentication)

	// iterating over map
	for k := range MetricGlobal.AuthRolesClients.Iter() {
		util.Logger.Debugf("Map contains key value: %s \t %u", (k.Key).(string), *((k.Value).(*uint64)))
		disMtr.AuthRolesClients[(k.Key).(string)] = *((k.Value).(*uint64))
	}
	for k := range MetricGlobal.Authentication.Iter() {
		var amt displayAuthentication
		amt.Rejected = *((k.Value).(*metricAuthentication)).Rejected
		amt.Succeded = *((k.Value).(*metricAuthentication)).Succeded
		disMtr.Authentication[(k.Key).(string)] = amt
	}

	return
}
