package filter

import (
	"github.com/gammazero/nexus/router"
	"github.com/gammazero/nexus/wamp"
)

const (
	KeyType    = "filter_type"
	KeyFilter  = "filter"
	KeyFilters = "filters"
	TypeNeg    = "not"
	TypeAny    = "any"
	TypeAll    = "all"
)

type complexFilter struct {
	match      string
	subFilters []router.PublishFilter
}

type negFilter struct {
	subFilter router.PublishFilter
}

func NewComplexFilter(msg *wamp.Publish) router.PublishFilter {
	x := createFilter(msg.Options)
	return x
}

func IsValidFilter(ftype string) bool {
	return ftype == TypeAll || ftype == TypeAny || ftype == TypeNeg
}

func createFilter(opts wamp.Dict) router.PublishFilter {
	if len(opts) == 0 {
		return nil
	}
	filters, fok := wamp.AsList(opts[KeyFilters])
	if ftype, ok := wamp.AsString(opts[KeyType]); !ok || !fok || !IsValidFilter(ftype) {
		return NewSimplePublishFilter(opts)
	} else if ftype == TypeNeg {
		filter, fok := wamp.AsDict(opts[KeyFilter])
		var subFilter router.PublishFilter
		if !fok {
			subFilter = NewSimplePublishFilter(opts)
		} else {
			subFilter = createFilter(filter)
		}
		if subFilter == nil {
			return nil
		}
		return &negFilter{
			subFilter: subFilter,
		}
	} else {
		subfilters := []router.PublishFilter{}
		for _, rawFilter := range filters {
			filterDict, ok := wamp.AsDict(rawFilter)
			if !ok {
				continue
			}
			filterObj := createFilter(filterDict)
			if filterObj == nil {
				continue
			}
			subfilters = append(subfilters, filterObj)
		}
		return &complexFilter{
			match:      ftype,
			subFilters: subfilters,
		}
	}
}

func (c *complexFilter) LockRequired() bool {
	for _, f := range c.subFilters {
		if f.LockRequired() {
			return true
		}
	}
	return false
}

func (n *negFilter) LockRequired() bool {
	return n.subFilter.LockRequired()
}

func (c *complexFilter) PublishAllowed(sub *wamp.Session) bool {
	if c.match == TypeAll {
		for _, f := range c.subFilters {
			if !f.PublishAllowed(sub) {
				return false
			}
		}
		return true
	}
	for _, f := range c.subFilters {
		if f.PublishAllowed(sub) {
			return true
		}
	}
	return false
}

func (n *negFilter) PublishAllowed(sub *wamp.Session) bool {
	return !n.subFilter.PublishAllowed(sub)
}
