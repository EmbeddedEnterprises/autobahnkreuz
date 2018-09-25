package cli

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"strings"

	"github.com/EmbeddedEnterprises/autobahnkreuz/util"

	flag "github.com/spf13/pflag"
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

type CLIParameters struct {
	ListenTLS *TLSEndpoint
	ListenWS  *WSEndpoint
	Realm     string

	EnableTicketAuth         bool
	UpstreamAuthFunc         string
	UpstreamGetAuthRolesFunc string
	ReservedAuthRole         []string
	EnableResumeToken        bool
	EnableMetrics            bool
	MetricPort               uint16

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

func ParseCLI() CLIParameters {
	// Step 1: Build the command line interface.
	// we start with some really basic parameters
	cliRealm := flag.StringP("realm", "r", "", "The realm to run the router on")
	cliAnonEnable := flag.Bool("enable-anonymous", true, "Whether to allow authmethod 'anonymous'")
	cliAnonRole := flag.String("anonymous-authrole", "anonymous", "Authentication role to assign to anonymous clients")
	cliTicketEnable := flag.Bool("enable-ticket", true, "Whether to allow authmethod 'ticket'")
	cliTicketUpstream := flag.String("ticket-check-func", "", "Which WAMP RPC to call when ticket authentication is requested")
	cliTicketRoleUpstream := flag.String("ticket-get-role-func", "", "Which WAMP RPC to call to resolve authid to authrole/authextra")
	cliExcludeAuthRoles := flag.StringSlice("exclude-auth-role", nil, "Authentication roles to exclude from ticket authentication")
	cliEnableResumeToken := flag.Bool("enable-resume-token", true, "Whether to allow ticket authentication to have a keep-me-logged-in token.")
	cliEnableMetricRecord := flag.Bool("enable-metrics", false, "Whether to expose the recorded metrics")
	cliMetricPort := flag.Uint16("metric-port", 7070, "Which port shall be used to expose the metrics api")

	// Unencrypted endpoint
	cliEnableWS := flag.Bool("enable-ws", true, "Enable unencrypted WebSocket endpoint")
	cliPortWS := flag.Uint16("ws-port", 8001, "Port for the WebSocket endpoint")
	cliHostWS := flag.String("ws-host", "", "Listen address for the WebSocket endpoint")

	// Encrypted endpoint
	cliEnableWSS := flag.Bool("enable-wss", true, "Enable encrypted WebSocket endpoint")
	cliPortWSS := flag.Uint16("wss-port", 8000, "Port for the TLS endpoint")
	cliHostWSS := flag.String("wss-host", "", "Listen address for the TLS endpoint")
	cliKeyWSS := flag.String("wss-key-file", "", "TLS Key file")
	cliCertWSS := flag.String("wss-cert-file", "", "TLS Cert file")
	cliClientAuth := flag.String("wss-client-auth", "accept", "Use TLS client authentication (values: 'no', 'accept', 'require')")
	cliClientCAs := flag.StringSlice("wss-client-ca", nil, "Acceptable client CAs and their authroles (format: 'authrole;ca-cert.pem,authrole;ca-cert.pem,....')")

	cliEnableAuthorizer := flag.Bool("enable-authorization", true, "Enable dynamic checking of auth roles")
	cliUpstreamAuthorizer := flag.String("authorizer-func", "", "Which WAMP RPC to call when an action has to be authorized")
	cliEnableFeatureAuthorizer := flag.Bool("enable-feature-authorization", false, "Enable authorization checking based on a feature matrix")
	cliUpstreamFeatureAuthorizerMatrix := flag.String("feature-authorizer-matrix-func", "", "Which WAMP RPC to call to get a feature matrix")
	cliUpstreamFeatureAuthorizerMapping := flag.String("feature-authorizer-mapping-func", "", "Which WAMP RPC to call to get a feature mapping")
	cliTrustAuthRoles := flag.StringSlice("trusted-authroles", nil, "Authorize any actions of these authentication roles")
	cliAuthorizerFailed := flag.String("authorizer-fallback", "reject", "Whether to permit any actions if the authorizer endpoint fails (values: 'permit', 'reject')")
	// Call the command line parser
	flag.Parse()

	config := CLIParameters{
		Realm:                    *cliRealm,
		EnableAnonymousAuth:      *cliAnonEnable,
		AnonymousAuthRole:        *cliAnonRole,
		EnableTicketAuth:         *cliTicketEnable,
		UpstreamAuthFunc:         *cliTicketUpstream,
		UpstreamGetAuthRolesFunc: *cliTicketRoleUpstream,
		EnableResumeToken:        *cliEnableResumeToken,
		ReservedAuthRole:         *cliExcludeAuthRoles,
		EnableAuthorizer:         *cliEnableAuthorizer,
		TrustedAuthRoles:         *cliTrustAuthRoles,
		UpstreamAuthorizer:       *cliUpstreamAuthorizer,
		EnableMetrics:            *cliEnableMetricRecord,
		MetricPort:               *cliMetricPort,

		EnableFeatureAuthorizer:          *cliEnableFeatureAuthorizer,
		UpstreamFeatureAuthorizerMatrix:  *cliUpstreamFeatureAuthorizerMatrix,
		UpstreamFeatureAuthorizerMapping: *cliUpstreamFeatureAuthorizerMapping,
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
		switch *cliAuthorizerFailed {
		case "permit", "accept":
			config.AuthorizeFailed = PermitAction
		default:
			config.AuthorizeFailed = RejectAction
		}
	}

	enabled := false
	if *cliEnableWS {
		enabled = true
		config.ListenWS = &WSEndpoint{
			Port: *cliPortWS,
			// Host may be empty here, which means 0.0.0.0
			Host: *cliHostWS,
		}
	}
	if *cliEnableWSS {
		enabled = true
		config.ListenTLS = &TLSEndpoint{
			WS: WSEndpoint{
				Port: *cliPortWSS,
				// Host may be empty here, which means 0.0.0.0
				Host: *cliHostWSS,
			},
		}

		serverCert, err := tls.LoadX509KeyPair(*cliCertWSS, *cliKeyWSS)
		if err != nil {
			util.Logger.Criticalf("Failed to load server certificate: %v", err)
			os.Exit(util.ExitArgument)
		}
		config.ListenTLS.Certificate = serverCert

		switch *cliClientAuth {
		case "no":
			config.ListenTLS.ClientCertPolicy = DisableClientAuthentication
		case "require":
			config.ListenTLS.ClientCertPolicy = RequireClientCert
		default:
			config.ListenTLS.ClientCertPolicy = AcceptClientCert
		}

		if config.ListenTLS.ClientCertPolicy != DisableClientAuthentication {
			for _, cca := range *cliClientCAs {
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
