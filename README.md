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

## About this project

There are multiple wamp routers available off the shelf, we evaluated many of them
but nothing fitted our requirements, so we decided to create our own router.

The name is deducted from the client libraries such as autobahn.js and autobahn|cpp.

Currently there is a lack of documentation, but we are working to fix this.
