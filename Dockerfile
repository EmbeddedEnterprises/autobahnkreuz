FROM golang:1.12-alpine as builder
RUN apk update && apk add build-base git

RUN mkdir -p $GOPATH/src/github.com/EmbeddedEnterprises/autobahnkreuz

COPY . $GOPATH/src/github.com/EmbeddedEnterprises/autobahnkreuz
WORKDIR $GOPATH/src/github.com/EmbeddedEnterprises/autobahnkreuz

RUN go get
RUN go build -o bin/autobahnkreuz -ldflags "-linkmode external -extldflags -static" -a main.go
RUN cp bin/autobahnkreuz /bin

FROM scratch
LABEL service "autobahnkreuz"
LABEL vendor "EmbeddedEnterprises"
LABEL maintainers "Martin Koppehel <mkoppehel@embedded.enterprises>"

COPY --from=builder /bin/autobahnkreuz /bin/autobahnkreuz
ENTRYPOINT ["/bin/autobahnkreuz"]
CMD []
