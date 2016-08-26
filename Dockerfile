FROM scratch
MAINTAINER Allegro
ADD bin/marathon-consul marathon-consul
EXPOSE 4000
ENTRYPOINT ["/marathon-consul"]