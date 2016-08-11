FROM gliderlabs/alpine:3.2
MAINTAINER Brian Hicks <brian@brianthicks.com>

RUN apk add --update ca-certificates bash
COPY launch.sh /launch.sh

COPY . /go/src/github.com/allegro/marathon-consul
RUN apk add make go git mercurial \
  && cd /go/src/github.com/allegro/marathon-consul \
  && export GOPATH=/go \
  && make build_no_test \
  && mv /go/src/github.com/allegro/marathon-consul/bin/marathon-consul /bin/marathon-consul \
  && rm -rf /go \
  && apk del --purge go git mercurial

EXPOSE 4000

ENTRYPOINT ["/launch.sh"]
