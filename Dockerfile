FROM embeddedenterprises/burrow as builder
RUN apk update && apk add build-base
RUN burrow clone https://github.com/EmbeddedEnterprises/autobahnkreuz.git
WORKDIR $GOPATH/src/github.com/EmbeddedEnterprises/autobahnkreuz
RUN burrow e && burrow b
RUN cp bin/autobahnkreuz /bin

FROM scratch
LABEL service "autobahnkreuz"
LABEL vendor "EmbeddedEnterprises"
LABEL maintainers "Martin Koppehel <mkoppehel@embedded.enterprises>"

COPY --from=builder /bin/autobahnkreuz /bin/autobahnkreuz
ENTRYPOINT ["/bin/autobahnkreuz"]
CMD []

