package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/gammazero/nexus/client"
	"github.com/gammazero/nexus/router"
	"github.com/op/go-logging"
)

const (
	ExitSuccess = iota
	ExitArgument
	ExitRunning
	ExitService
)

// EnvLogFormat is kept for compatibility with the service lib.
const EnvLogFormat string = "SERVICE_LOGFORMAT"

// EnvLogLevel determines the desired level of logging depth
const EnvLogLevel string = "SERVICE_LOGLEVEL"

const ModuleName string = "enterprises.embedded.autobahnkreuz"

var Logger *logging.Logger
var LocalClient *client.Client
var Router router.Router
var DebugRouter bool

func init() {
	// setup logging library
	var err error
	Logger, err = logging.GetLogger(ModuleName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %s\n", err)
		os.Exit(ExitService)
	}

	// write to Stderr to keep Stdout free for data output
	backend := logging.NewLogBackend(os.Stderr, "", 0)

	// read an environment variable controlling the log format
	// possibilities are "k8s" or "cluster" or "machine" for a machine readable format
	// and "debug" or "human" for a human readable format (default)
	// the values are case insensitive
	var logFormat logging.Formatter
	envLogFormat := strings.ToLower(os.Getenv(EnvLogFormat))
	switch envLogFormat {
	case "", "human", "debug":
		logFormat, err = logging.NewStringFormatter(`%{color}[%{level:-8s}] %{time:15:04:05.000} %{longpkg}@%{shortfile}%{color:reset} -- %{message}`)
	case "k8s", "cluster", "machine":
		logFormat, err = logging.NewStringFormatter(`[%{level:-8s}] %{time:2006-01-02T15:04:05.000} %{shortfunc} -- %{message}`)
	default:
		Logger.Criticalf("Failed to setup log format: invalid format %s", envLogFormat)
		os.Exit(ExitArgument)
	}
	if err != nil {
		Logger.Criticalf("Failed to create logging format, shutting down: %s", err)
		os.Exit(ExitArgument)
	}

	backendFormatted := logging.NewBackendFormatter(backend, logFormat)
	logging.SetBackend(backendFormatted)

	// read environment variable of level
	var logLevel logging.Level
	switch strings.ToUpper(os.Getenv(EnvLogLevel)) {
	case "CRITICAL":
		logLevel = logging.CRITICAL
	case "ERROR":
		logLevel = logging.ERROR
	case "WARN":
		logLevel = logging.WARNING
	case "INFO":
		logLevel = logging.INFO
	case "DEBUG":
		logLevel = logging.DEBUG
		DebugRouter = true
	default:
		logLevel = logging.INFO
	}
	// since we only use one logging backend and have the name on hand this will suffice
	logging.SetLevel(logLevel, ModuleName)
}
