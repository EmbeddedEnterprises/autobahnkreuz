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
	InMessageCount         *uint64
	OutMessageCount        *uint64
	InTrafficBytesTotal    *uint64
	OutTrafficBytesTotal   *uint64
	Authentication         *hashmap.HashMap
	AuthRolesClients       *hashmap.HashMap
	SucceededAuthorization *uint64
	RejectedAuthorization  *uint64
}

type metricAuthentication struct {
	Succeeded *uint64
	Rejected  *uint64
}

type displayAuthentication struct {
	Succeeded uint64
	Rejected  uint64
}

type displayGeneral struct {
	RecvMessageCount       uint64
	SendMessageCount       uint64
	RecvTrafficBytesTotal  uint64
	SendTrafficBytesTotal  uint64
	Authentication         map[string]displayAuthentication
	AuthRolesClients       map[string]uint64
	SucceededAuthorization uint64
	RejectedAuthorization  uint64
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
		InMessageCount:         new(uint64),
		OutMessageCount:        new(uint64),
		InTrafficBytesTotal:    new(uint64),
		AuthRolesClients:       hashmap.New(128),
		Authentication:         hashmap.New(128),
		OutTrafficBytesTotal:   new(uint64),
		SucceededAuthorization: new(uint64),
		RejectedAuthorization:  new(uint64),
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

func IncreaseAtomic(value *uint64, diff uint64) {
	atomic.AddUint64(value, diff)
}

// ConditionalIncrement is only used in combination with Succeeded or Rejected authorization so no extra catch there
func ConditionalIncrement(permit bool) {
	if permit {
		IncrementAtomic(MetricGlobal.SucceededAuthorization)
	} else {
		IncrementAtomic(MetricGlobal.RejectedAuthorization)
	}
}

// IncrementAuth atomic increases the counter of a certain authentication method `name` depended on if the authentication succeeded or not
func IncrementAuth(name string, succeeded bool) {
	var amt metricAuthentication
	curamt, loaded := MetricGlobal.Authentication.GetOrInsert(name, &amt)
	mtrA := ((curamt).(*metricAuthentication))
	util.Logger.Debugf("Authentication Method registered: %s", name)
	if !loaded {
		util.Logger.Debugf("Authentication Values created for %s", name)
		mtrA.Succeeded = new(uint64)
		mtrA.Rejected = new(uint64)
	}
	if succeeded {
		atomic.AddUint64(mtrA.Succeeded, 1)
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
	util.Logger.Debug("SendHandler ping")
	IncrementAtomic(MetricGlobal.OutMessageCount)
}

func RecvHandler() {
	// util.Logger.Debug("RecvHandler ping")
	IncrementAtomic(MetricGlobal.InMessageCount)
}

func RecvMsgLenHandler(len uint64) {
	// util.Logger.Debugf("Received message of length: %d", len)
	// util.Logger.Debugf("Current total before adding: %d", *MetricGlobal.InTrafficBytesTotal)
	IncreaseAtomic(MetricGlobal.InTrafficBytesTotal, len)
	// util.Logger.Debugf("Current total after adding: %d", *MetricGlobal.InTrafficBytesTotal)
}

func SendMsgLenHandler(len uint64) {
	// util.Logger.Debugf("Send message of length: %d", len)
	IncreaseAtomic(MetricGlobal.OutTrafficBytesTotal, len)
}

func processMtr() (disMtr displayGeneral, err error) {
	// Setting all single valued fields
	disMtr.RecvMessageCount = *MetricGlobal.InMessageCount
	disMtr.SendMessageCount = *MetricGlobal.OutMessageCount
	disMtr.RejectedAuthorization = *MetricGlobal.RejectedAuthorization
	disMtr.SucceededAuthorization = *MetricGlobal.SucceededAuthorization
	disMtr.RecvTrafficBytesTotal = *MetricGlobal.InTrafficBytesTotal
	disMtr.SendTrafficBytesTotal = *MetricGlobal.OutTrafficBytesTotal

	// initialize maps
	disMtr.AuthRolesClients = make(map[string]uint64)
	disMtr.Authentication = make(map[string]displayAuthentication)

	// iterating over map
	for k := range MetricGlobal.AuthRolesClients.Iter() {
		util.Logger.Debugf("Map contains key value: %s \t %d", (k.Key).(string), *((k.Value).(*uint64)))
		disMtr.AuthRolesClients[(k.Key).(string)] = *((k.Value).(*uint64))
	}
	for k := range MetricGlobal.Authentication.Iter() {
		var amt displayAuthentication
		amt.Rejected = *((k.Value).(*metricAuthentication).Rejected)
		amt.Succeeded = *((k.Value).(*metricAuthentication).Succeeded)
		disMtr.Authentication[(k.Key).(string)] = amt
	}

	return
}
