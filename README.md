# autobahnkreuz [![Latest Tag](https://img.shields.io/github/tag/EmbeddedEnterprises/autobahnkreuz.svg)](https://github.com/EmbeddedEnterprises/autobahnkreuz/releases) [![Go Report Card](https://goreportcard.com/badge/github.com/EmbeddedEnterprises/autobahnkreuz)](https://goreportcard.com/report/github.com/EmbeddedEnterprises/autobahnkreuz) [![GoDoc](https://godoc.org/github.com/EmbeddedEnterprises/autobahnkreuz?status.svg)](https://godoc.org/github.com/EmbeddedEnterprises/autobahnkreuz) [![Docker Pulls](https://img.shields.io/docker/pulls/embeddedenterprises/autobahnkreuz.svg)](https://hub.docker.com/r/embeddedenterprises/autobahnkreuz/) [![Docker Build Status](https://img.shields.io/docker/build/embeddedenterprises/autobahnkreuz.svg)](https://hub.docker.com/r/embeddedenterprises/autobahnkreuz/builds/) [![Docker Image Size](https://img.shields.io/microbadger/image-size/embeddedenterprises/autobahnkreuz.svg)](https://hub.docker.com/r/embeddedenterprises/autobahnkreuz/)  [![Liberapay](https://img.shields.io/liberapay/receives/embeddedenterprises.svg?logo=liberapay)](https://liberapay.com/EmbeddedEnterprises/donate)


An advanced wamp router based on nexus.

## Installation

### Docker

[Docker Hub](https://hub.docker.com/r/embeddedenterprises/autobahnkreuz/)

You can easily get an docker image on your machine.  
`docker pull embeddedenterprises/autobahnkreuz:edge`

You can start and configure this image with the following command.  
`docker run embeddedenterprises/autobahnkreuz:edge`

Afterwards you can enter your configuration parameters. The entire router does not save state to the container or to mounted volumes.

### Local

It is nessesary to have a working go installation on your system.  
We recommend to use [burrow](https://github.com/EmbeddedEnterprises/burrow) to install this project.  
Burrow is another go dependency management tool, which is also maintainted by [EmbeddedEnterprises](https://github.com/EmbeddedEnterprises).

`burrow clone https://github.com/EmbeddedEnterprises/autobahnkreuz/` installs the programm to the correct location in your go path.
After entering the `autobahnkreuz` folder, you can use `burrow run` to start `autobahnkreuz`.
To configure the instance, you can append your configuration parameters to the `burrow run` command, e.g. `burrow run -- -h.`

## Configuration

Short Command | Long Command | Parameter | Description | Reference
-----|--------|--------------|-----------|-------------|----------------------------------------------------------------------------------------------------------
     | --anonymous-authrole | string              | Authentication role to assign to anonymous clients (default "anonymous") | Foo
     | --authorizer-fallback | string             | Whether to permit any actions if the authorizer endpoint fails (values: 'permit', 'reject') (default "reject")
     | --authorizer-func | string                 | Which WAMP RPC to call when an action has to be authorized
     | --enable-anonymous |                        | Whether to allow authmethod 'anonymous' (default true)
     | --enable-authorization |                     | Enable dynamic checking of auth roles (default true)
     | --enable-feature-authorization |             | Enable authorization checking based on a feature matrix
     | --enable-resume-token |           | Whether to allow ticket authentication to have a keep-me-logged-in token. (default true)
     | --enable-ticket |                            | Whether to allow authmethod 'ticket' (default true)
     | --enable-ws |                               | Enable unencrypted WebSocket endpoint (default true)
     | --enable-wss |                              | Enable encrypted WebSocket endpoint (default true)
     | --exclude-auth-role | strings              | Authentication roles to exclude from ticket authentication
     | --feature-authorizer-mapping-func | string | Which WAMP RPC to call to get a feature mapping
     | --feature-authorizer-matrix-func | string  | Which WAMP RPC to call to get a feature matrix
 -r  | --realm | string                           | The realm to run the router on
     | --ticket-check-func | string               | Which WAMP RPC to call when ticket authentication is requested
     | --ticket-get-role-func | string            | Which WAMP RPC to call to resolve authid to authrole/authextra
     | --trusted-authroles | strings              | Authorize any actions of these authentication roles
     | --ws-host | string                         | Listen address for the WebSocket endpoint
     | --ws-port | uint16                         | Port for the WebSocket endpoint (default 8001)
     | --wss-cert-file | string                   | TLS Cert file
     | --wss-client-auth | string                   | Use TLS client authentication (values: 'no', 'accept', 'require') (default "accept")
     | --wss-client-ca | strings                  | Acceptable client CAs and their authroles (format: 'authrole;ca-cert.pem,authrole;ca-cert.pem,....')
     | --wss-host | string                        | Listen address for the TLS endpoint
     | --wss-key-file | string                    | TLS Key file
     | --wss-port | uint16                        | Port for the TLS endpoint (default 8000)

### WS

The simplest way to connect to `autobahnkreuz` are websockets. You can use client libraries like [nexus](https://github.com/gammarzero/nexus) or [autobahn.js](https://github.com/crossbario/autobahn-js). But `autobahnkreuz` is designed to work with high-level service libraries, which are called service libs.

We provide `service` libraries for different languages.

+ [Node](https://github.com/creatdevsolutions/service)
+ [go](https://github.com/EmbeddedEnterprises/service)

### WSS

## About this project

There are multiple wamp routers available off the shelf, we evaluated many of them
but nothing fitted our requirements, so we decided to create our own router.

The name is deducted from the client libraries such as autobahn.js and autobahn|cpp.

Currently there is a lack of documentation, but we are working to fix this.
