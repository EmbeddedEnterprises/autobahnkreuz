FROM golang:1.12-alpine as builder
RUN apk update && apk add build-base git

RUN mkdir -p /autobahnkreuz

COPY . /autobahnkreuz
WORKDIR /autobahnkreuz

RUN go get
RUN go build -o bin/autobahnkreuz -ldflags "-linkmode external -extldflags -static" -a main.go

FROM scratch
LABEL service "autobahnkreuz"
LABEL vendor "EmbeddedEnterprises"
LABEL maintainers "Martin Koppehel <mkoppehel@embedded.enterprises>"

COPY --from=builder /autobahnkreuz/bin/autobahnkreuz /bin/autobahnkreuz
ENTRYPOINT ["/bin/autobahnkreuz"]
CMD []
