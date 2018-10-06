package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/EmbeddedEnterprises/autobahnkreuz/auth"
	"github.com/EmbeddedEnterprises/autobahnkreuz/cli"
	"github.com/EmbeddedEnterprises/autobahnkreuz/filter"
	"github.com/EmbeddedEnterprises/autobahnkreuz/metrics"
	"github.com/EmbeddedEnterprises/autobahnkreuz/ping"
	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	"github.com/deckarep/golang-set"
	"github.com/gammazero/nexus/client"
	"github.com/gammazero/nexus/router"
	"github.com/gammazero/nexus/transport"
	"github.com/gammazero/nexus/transport/serialize"
	"github.com/gammazero/nexus/wamp"
)

type Initializer func()

func verifyPeer(requireCert bool, validCAs *x509.CertPool) func([][]byte, [][]*x509.Certificate) error {

	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if !requireCert && len(rawCerts) == 0 {
			// no error, because other authmethods are permittable
			return nil
		}

		for i := 0; i < len(rawCerts); i++ {
			certASN1 := rawCerts[i]
			cert, err := x509.ParseCertificates(certASN1)
			if err != nil {
				util.Logger.Warningf("Failed to parse client certificate: %v", err)
				return err
			}

			if len(cert) != 1 {
				return errors.New("Client supplied bogus certificate.")
			}

			if cert[0].IsCA {
				continue
			}

			usage := []x509.ExtKeyUsage{
				x509.ExtKeyUsageClientAuth,
			}
			_, err = cert[0].Verify(x509.VerifyOptions{
				DNSName:       "",
				Intermediates: nil,
				Roots:         validCAs,
				KeyUsages:     usage,
			})
			if err != nil {
				util.Logger.Warningf("Client certificate validation failed: %v", err)
			} else {
				return nil
			}
		}
		return errors.New("Failed to verify client certificate: No provided certificates matched client CA")
	}
}

func createRouterConfig(config cli.CLIParameters) (*router.RouterConfig, []Initializer) {
	encode := func(value reflect.Value) ([]byte, error) {
		return value.Bytes(), nil
	}
	decode := func(value reflect.Value, data []byte) error {
		value.Elem().SetBytes(data)
		return nil
	}
	serialize.MsgpackRegisterExtension(reflect.TypeOf(serialize.BinaryData{}), 42, encode, decode)
	// Create router instance.
	routerConfig := &router.RouterConfig{}
	realm := &router.RealmConfig{
		URI:           wamp.URI(config.Realm),
		AnonymousAuth: false,
		// This is required for localPeers to work.
		RequireLocalAuth:     false,
		AllowDisclose:        true,
		EnableMetaKill:       true,
		EnableMetaModify:     true,
		PublishFilterFactory: filter.NewComplexFilter,
	}
	var initers []Initializer

	routerConfig.RealmConfigs = []*router.RealmConfig{
		realm,
	}

	if config.EnableAnonymousAuth {
		util.Logger.Infof("Enabling anonymous authentication, role: %v", config.AnonymousAuthRole)
		realm.Authenticators = append(realm.Authenticators, auth.AnonymousAuth{
			AuthRole: config.AnonymousAuthRole,
		})
	}

	exclude := mapset.NewSet()
	for _, x := range config.ReservedAuthRole {
		exclude.Add(x)
	}

	if config.EnableTicketAuth {
		util.Logger.Infof("Enabling ticket auth, func: %v, roles: %v", config.UpstreamAuthFunc, config.UpstreamGetAuthRolesFunc)
		authenticator, err := auth.NewDynamicTicket(config.UpstreamAuthFunc, config.UpstreamGetAuthRolesFunc, config.Realm, exclude, config.EnableResumeToken)
		if err != nil {
			util.Logger.Criticalf("Failed to create dynamic ticket authenticator: %v", err)
			os.Exit(1)
		}
		realm.Authenticators = append(realm.Authenticators, authenticator)
	}

	if config.EnableResumeToken {
		util.Logger.Infof("Enabling resume token auth, roles: %v", config.UpstreamGetAuthRolesFunc)
		authenticator, err := auth.NewResumeAuthenticator(config.UpstreamGetAuthRolesFunc, config.Realm, exclude)
		if err != nil {
			util.Logger.Criticalf("Failed to create resume authenticator: %v", err)
		}
		realm.Authenticators = append(realm.Authenticators, authenticator)
		initers = append(initers, authenticator.Initialize)
	}

	if config.EnableAuthorizer || config.EnableFeatureAuthorizer {

		trustedAuthRoles := mapset.NewSet()
		for _, tAuthRole := range config.TrustedAuthRoles {
			trustedAuthRoles.Add(tAuthRole)
		}
		// This is required to make the authorizer and authenticator working.
		// If you use client.ConnectLocal, the client automagically gets the trusted auth role.
		trustedAuthRoles.Add("trusted")

		if config.EnableAuthorizer {
			util.Logger.Infof("Enabling authorization, func: %v", config.UpstreamAuthorizer)

			realm.Authorizer = auth.DynamicAuthorizer{
				UpstreamAuthorizer: config.UpstreamAuthorizer,
				TrustedAuthRoles:   trustedAuthRoles,
				PermitDefault:      config.AuthorizeFailed == cli.PermitAction,
			}
		} else if config.EnableFeatureAuthorizer {
			util.Logger.Infof("Enabling Feature Authorization.")
			util.Logger.Infof("Ensure, you really want to do this. Dynamic Authorization is not active.")

			authRef := auth.NewFeatureAuthorizer(
				config.AuthorizeFailed == cli.PermitAction,
				config.UpstreamFeatureAuthorizerMatrix,
				config.UpstreamFeatureAuthorizerMapping,
				trustedAuthRoles,
			)

			realm.Authorizer = authRef
			initers = append(initers, authRef.Initialize)
		}
	}

	if config.ListenTLS != nil && config.ListenTLS.ClientCertPolicy != cli.DisableClientAuthentication {
		util.Logger.Infof("Enabling TLS client auth, %d valid client CAs", len(config.ListenTLS.ValidClientCAs))
		realm.Authenticators = append(realm.Authenticators, auth.TLSAuth{
			ValidClientCAs: config.ListenTLS.ValidClientCAs,
		})
	}
	return routerConfig, initers
}

func runTLSEndpoint(websocketServer *router.WebsocketServer, config cli.CLIParameters) io.Closer {
	if config.ListenTLS == nil {
		return nil
	}

	tlsCfg := &tls.Config{
		// We NEVER want to skip verification. It's dangerous.
		InsecureSkipVerify: false,
	}

	tlsCfg.Certificates = append(tlsCfg.Certificates, config.ListenTLS.Certificate)

	switch config.ListenTLS.ClientCertPolicy {
	case cli.DisableClientAuthentication:
		tlsCfg.ClientAuth = tls.NoClientCert
	case cli.AcceptClientCert:
		tlsCfg.ClientAuth = tls.VerifyClientCertIfGiven
	case cli.RequireClientCert:
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	tlsCfg.ClientCAs = x509.NewCertPool()
	for _, ca := range config.ListenTLS.ValidClientCAs {
		tlsCfg.ClientCAs.AddCert(ca.CACert)
	}
	requireCert := tlsCfg.ClientAuth == tls.RequireAndVerifyClientCert

	tlsCfg.VerifyPeerCertificate = verifyPeer(requireCert, tlsCfg.ClientCAs)
	// Create and run server.

	closer, err := websocketServer.ListenAndServeTLS(fmt.Sprintf(
		"%s:%d",
		config.ListenTLS.WS.Host,
		config.ListenTLS.WS.Port,
	), tlsCfg, "", "")
	if err != nil {
		util.Logger.Criticalf("ListenAndServeTLS failed: %v", err)
		os.Exit(1)
	}
	return closer
}

func runWSEndpoint(websocketServer *router.WebsocketServer, config cli.CLIParameters) io.Closer {
	if config.ListenWS == nil {
		return nil
	}

	closer, err := websocketServer.ListenAndServe(fmt.Sprintf(
		"%s:%d",
		config.ListenWS.Host,
		config.ListenWS.Port,
	))
	if err != nil {
		util.Logger.Criticalf("ListenAndServe failed: %v", err)
		os.Exit(1)
	}
	return closer

}

func generateWebsocketServer(nxr *router.Router) *router.WebsocketServer {
	// Create and run server.
	srv := router.NewWebsocketServer(*nxr, metrics.SendMsgLenHandler, metrics.RecvMsgLenHandler, metrics.SendHandler, metrics.RecvHandler)
	srv.SetConfig(transport.WebsocketConfig{
		EnableRequestCapture: true,
		SendCallback:         metrics.SendHandler,
		RecvCallback:         metrics.RecvHandler,
		InMsgLenCallback:     metrics.RecvMsgLenHandler,
		OutMsgLenCallback:    metrics.SendMsgLenHandler,
	})

	srv.KeepAlive = 5 * time.Second

	// Disable CORS, since we're running behind a reverse proxy anywayW
	srv.Upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	return srv
}

func main() {
	var err error
	util.Init()
	util.Logger.Debug("Interconnect startup")
	config := cli.ParseCLI()

	// starting up metrics
	metrics.Init(config.MetricPort, config.EnableMetrics, false)
	if err != nil {
		util.Logger.Criticalf("Failed to start metrics: %v", err)
		os.Exit(util.ExitService)
	}

	routerConfig, initers := createRouterConfig(config)

	util.Router, err = router.NewRouter(routerConfig, nil)
	if err != nil {
		util.Logger.Criticalf("Failed to start router: %v", err)
		os.Exit(util.ExitService)
	}
	defer util.Router.Close()

	websocketServer := generateWebsocketServer(&util.Router)

	closerTLS := runTLSEndpoint(websocketServer, config)
	closer := runWSEndpoint(websocketServer, config)

	util.LocalClient, err = client.ConnectLocal(util.Router, client.ClientConfig{
		Realm: config.Realm,
	})
	if err != nil {
		util.Logger.Criticalf("Failed to connect local client: %v", err)
		os.Exit(1)
	}
	if err := ping.RegisterPing(util.LocalClient); err != nil {
		util.Logger.Criticalf("Failed to register ping function!")
		os.Exit(1)
	}

	util.Logger.Info("Router started, local client connected.")

	// Wait for SIGINT (CTRL-c), then close server and exit.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)
	go func() {
		for _, initer := range initers {
			initer()
		}
	}()
	<-shutdown

	util.Logger.Info("SIGINT received, terminating.")

	if closerTLS != nil {
		closerTLS.Close()
	}

	if closer != nil {
		closer.Close()
	}
}
