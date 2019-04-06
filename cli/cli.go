package cli

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/flags"
	"io/ioutil"
	"os"
	"strings"

	"github.com/EmbeddedEnterprises/autobahnkreuz/util"
)

type TLSClientCAInfo struct {
	AuthRole string
	CACert   *x509.Certificate
}

type CertificatePolicy int

const (
	DisableClientAuthentication CertificatePolicy = iota
	AcceptClientCert
	RequireClientCert
)

type AuthorizerMissingPolicy int

const (
	RejectAction AuthorizerMissingPolicy = iota
	PermitAction
)

type TLSEndpoint struct {
	WS               WSEndpoint
	Certificate      tls.Certificate
	ClientCertPolicy CertificatePolicy
	ValidClientCAs   []TLSClientCAInfo
}

type WSEndpoint struct {
	Port uint16
	Host string
}

type InterconnectConfiguration struct {
	ListenTLS *TLSEndpoint
	ListenWS  *WSEndpoint
	Realm     string

	EnableTicketAuth         bool
	UpstreamAuthFunc         string
	UpstreamGetAuthRolesFunc string
	ReservedAuthRole         []string
	EnableResumeToken        bool

	EnableAnonymousAuth bool
	AnonymousAuthRole   string

	// Global Authorization Variables
	// Works in both authenticators
	TrustedAuthRoles []string
	AuthorizeFailed  AuthorizerMissingPolicy

	// Dynamic Authorizer
	// According to wamp-proto
	EnableAuthorizer   bool
	UpstreamAuthorizer string

	// Feature Authorizer
	// According to my brain and my whiteboard
	EnableFeatureAuthorizer          bool
	UpstreamFeatureAuthorizerMatrix  string
	UpstreamFeatureAuthorizerMapping string
}

type Configuration struct {
	Realm             string   `config:"realm, required"`
	EnableAnonymous   bool     `config:"enable-anonymous"`
	AnonymousAuthRole string   `config:"anonymous-authrole"`
	EnableTicket      bool     `config:"enable-ticket"`
	TicketCheckFunc   string   `config:"ticket-check-func"`
	TicketGetRoleFunc string   `config:"ticket-get-role-func"`
	ExcludeAuthRole   []string `config:"exclude-auth-role"`
	EnableResumeToken bool     `config:"enable-resume-token"`

	EnableWs bool   `config:"enable-ws"`
	WsHost   string `config:"ws-host"`
	WsPort   uint16 `config:"ws-port"`

	EnableWss     bool     `config:"enable-wss"`
	WssHost       string   `config:"wss-host"`
	WssPort       uint16   `config:"wss-port"`
	WssCertFile   string   `config:"wss-cert-file"`
	WssKeyFile    string   `config:"wss-key-file"`
	WssClientAuth string   `config:"wss-client-auth"`
	WssClientCA   []string `config:"wss-client-ca"`

	EnableAuthorizer                bool     `config:"enable-authorization"`
	AuthorizerFunc                  string   `config:"authorizer-func"`
	EnableFeatureAuthorization      bool     `config:"enable-feature-authorization"`
	FeatureAuthorizationMatrixFunc  string   `config:"feature-authorizer-matrix-func"`
	FeatureAuthorizationMappingFunc string   `config:"feature-authorizer-mapping-func"`
	TrustedAuthRoles                []string `config:"trusted-authroles"`
	AuthorizerFallback              string   `config:"authorizer-fallback"`
}

func assertNotEmpty(name, value string) {
	if value == "" {
		util.Logger.Criticalf("%s must not be empty!", name)
		os.Exit(util.ExitArgument)
	}
}

func parseClientCA(cca string) TLSClientCAInfo {
	x := strings.Split(cca, ";")
	if len(x) != 2 {
		util.Logger.Criticalf("ClientCA is in invalid format, expected authrole;ca-cert.pem, got: %s", cca)
		os.Exit(util.ExitArgument)
	}
	cert, err := ioutil.ReadFile(x[1])
	if err != nil {
		util.Logger.Criticalf("Failed to load client CA %s: %v", x[1], err)
		os.Exit(util.ExitArgument)
	}
	pem, _ := pem.Decode(cert)
	if pem == nil {
		util.Logger.Criticalf("Failed to parse PEM data of client CA %s", x[1])
		os.Exit(util.ExitArgument)
	}
	certObj, err := x509.ParseCertificate(pem.Bytes)
	if err != nil {
		util.Logger.Criticalf("Failed to parse client CA %s: %v", x[1], err)
		os.Exit(util.ExitArgument)
	}

	return TLSClientCAInfo{
		AuthRole: x[0],
		CACert:   certObj,
	}
}

func ParseCLI() InterconnectConfiguration {

	cliInput := Configuration{
		EnableAnonymous:   true,
		AnonymousAuthRole: "anonymous",
		EnableTicket:      true,
		EnableResumeToken: true,

		EnableWs: true,
		WsPort:   8001,

		EnableWss:     true,
		WssPort:       8000,
		WssClientAuth: "accept",

		EnableAuthorizer:           true,
		EnableFeatureAuthorization: true,
		AuthorizerFallback:         "reject",
	}

	loader := confita.NewLoader(
		flags.NewBackend(),
	)

	err := loader.Load(context.Background(), &cliInput)

	if err != nil {
		util.Logger.Critical("Failed to load configuration")
		util.Logger.Critical(err)
		os.Exit(util.ExitArgument)
	}

	config := InterconnectConfiguration{
		Realm:                    cliInput.Realm,
		EnableAnonymousAuth:      cliInput.EnableAnonymous,
		AnonymousAuthRole:        cliInput.AnonymousAuthRole,
		EnableTicketAuth:         cliInput.EnableTicket,
		UpstreamAuthFunc:         cliInput.TicketCheckFunc,
		UpstreamGetAuthRolesFunc: cliInput.TicketGetRoleFunc,
		EnableResumeToken:        cliInput.EnableResumeToken,
		ReservedAuthRole:         cliInput.ExcludeAuthRole,
		EnableAuthorizer:         cliInput.EnableAuthorizer,
		TrustedAuthRoles:         cliInput.TrustedAuthRoles,
		UpstreamAuthorizer:       cliInput.AuthorizerFunc,

		EnableFeatureAuthorizer:          cliInput.EnableFeatureAuthorization,
		UpstreamFeatureAuthorizerMatrix:  cliInput.FeatureAuthorizationMatrixFunc,
		UpstreamFeatureAuthorizerMapping: cliInput.FeatureAuthorizationMappingFunc,
	}

	assertNotEmpty("Realm", config.Realm)
	if config.EnableAnonymousAuth {
		assertNotEmpty("Anonymous authentication role", config.AnonymousAuthRole)
	}

	if config.EnableTicketAuth {
		assertNotEmpty("Ticket check function", config.UpstreamAuthFunc)
	}

	if config.EnableResumeToken || config.EnableTicketAuth {
		assertNotEmpty("Auth role getter function", config.UpstreamGetAuthRolesFunc)
	}

	if config.EnableAuthorizer && config.EnableFeatureAuthorizer {
		util.Logger.Criticalf("Can't enable both authorizers. Choose one!")
		os.Exit(util.ExitArgument)
	}

	if config.EnableFeatureAuthorizer {
		assertNotEmpty("Feature Authorizer Matrix", config.UpstreamFeatureAuthorizerMatrix)
		assertNotEmpty("Feature Authorizer Mapping", config.UpstreamFeatureAuthorizerMapping)
	}

	if config.EnableAuthorizer {
		assertNotEmpty("Authorization function", config.UpstreamAuthorizer)
		switch cliInput.AuthorizerFallback {
		case "permit", "accept":
			config.AuthorizeFailed = PermitAction
		default:
			config.AuthorizeFailed = RejectAction
		}
	}

	enabled := false
	if cliInput.EnableWs {
		enabled = true
		config.ListenWS = &WSEndpoint{
			Port: cliInput.WsPort,
			// Host may be empty here, which means 0.0.0.0
			Host: cliInput.WsHost,
		}
	}
	if cliInput.EnableWss {
		enabled = true
		config.ListenTLS = &TLSEndpoint{
			WS: WSEndpoint{
				Port: cliInput.WssPort,
				// Host may be empty here, which means 0.0.0.0
				Host: cliInput.WssHost,
			},
		}

		serverCert, err := tls.LoadX509KeyPair(cliInput.WssCertFile, cliInput.WssKeyFile)
		if err != nil {
			util.Logger.Criticalf("Failed to load server certificate: %v", err)
			os.Exit(util.ExitArgument)
		}
		config.ListenTLS.Certificate = serverCert

		switch cliInput.WssClientAuth {
		case "no":
			config.ListenTLS.ClientCertPolicy = DisableClientAuthentication
		case "require":
			config.ListenTLS.ClientCertPolicy = RequireClientCert
		default:
			config.ListenTLS.ClientCertPolicy = AcceptClientCert
		}

		if config.ListenTLS.ClientCertPolicy != DisableClientAuthentication {
			for _, cca := range cliInput.WssClientCA {
				config.ListenTLS.ValidClientCAs = append(
					config.ListenTLS.ValidClientCAs,
					parseClientCA(cca),
				)
			}
			if config.ListenTLS.ValidClientCAs == nil {
				util.Logger.Critical("You have to specify at least one client CA to authenticate against.")
				os.Exit(util.ExitArgument)
			}
		}
	}
	if !enabled {
		util.Logger.Critical("At least one transport must be enabled!")
		os.Exit(util.ExitArgument)
	}
	if !config.EnableTicketAuth && !config.EnableAnonymousAuth && (config.ListenTLS == nil || config.ListenTLS.ClientCertPolicy == DisableClientAuthentication) {
		util.Logger.Critical("You have to enable at least one authentication method!")
		util.Logger.Critical("Otherwise no client will be able to connect!")
		os.Exit(util.ExitArgument)
	}
	return config
}
