FROM alpine
MAINTAINER Brian Hicks <brian@brianthicks.com>

COPY . /go/src/github.com/CiscoCloud/marathon-forwarder
RUN apk add --update go git mercurial \
	&& cd /go/src/github.com/CiscoCloud/marathon-forwarder \
	&& export GOPATH=/go \
	&& go get \
	&& go build -o /bin/marathon-forwarder \
	&& rm -rf /go \
	&& apk del --purge go git mercurial

ENTRYPOINT ["/bin/marathon-forwarder"]
