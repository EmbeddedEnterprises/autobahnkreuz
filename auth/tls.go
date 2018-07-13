package auth

import (
	"bytes"
	"errors"
	"github.com/EmbeddedEnterprises/autobahnkreuz/cli"
	"github.com/EmbeddedEnterprises/autobahnkreuz/util"
	"net/http"

	"github.com/gammazero/nexus/wamp"
)

type TLSAuth struct {
	ValidClientCAs []cli.TLSClientCAInfo
}

func (self TLSAuth) Authenticate(sid wamp.ID, details wamp.Dict, client wamp.Peer) (*wamp.Welcome, error) {
	util.Logger.Debugf("TLS auth by sid: %v\n", sid)
	tpdet, ok := details["transport"].(wamp.Dict)
	if !ok {
		util.Logger.Error("No transport details given!")
		return nil, errors.New("Unauthorized")
	}
	authdet, ok := tpdet["auth"].(wamp.Dict)
	if !ok {
		util.Logger.Error("No auth key in transport details!")
		return nil, errors.New("Unauthorized")
	}
	req, ok := authdet["request"].(*http.Request)
	if !ok || req == nil || req.TLS == nil {
		util.Logger.Error("HTTP Request is broken.")
		return nil, errors.New("Unauthorized")
	}

	certs := req.TLS.PeerCertificates
	for _, ccert := range certs {
		if ccert.IsCA {
			util.Logger.Debugf("Skipping client supplied CA certificate: %v", ccert.Subject)
			continue
		}
		util.Logger.Debugf("Validating client cert: %v", ccert.Subject.CommonName)
		for _, cca := range self.ValidClientCAs {
			if bytes.Equal(ccert.RawIssuer, cca.CACert.RawSubject) {
				util.Logger.Debugf("Successful TLS auth by sid: %v, role: %v", sid, cca.AuthRole)
				return &wamp.Welcome{
					Details: wamp.Dict{
						"authid":     ccert.Subject.CommonName,
						"authmethod": self.AuthMethod(),
						"authrole": wamp.List{
							cca.AuthRole,
						},
						"authprovider": "static",
					},
				}, nil
			}
		}
	}

	return nil, errors.New("Unauthorized")
}

func (self TLSAuth) AuthMethod() string {
	return "tls"
}
