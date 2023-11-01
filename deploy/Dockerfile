FROM busybox@sha256:023917ec6a886d0e8e15f28fb543515a5fcd8d938edb091e8147db4efed388ee

LABEL org.opencontainers.image.source https://github.com/tkhq/traefik-plugin-geoblock

RUN mkdir /dist

COPY go.mod /dist/
COPY go.sum /dist/
COPY .traefik.yml /dist/
COPY LICENSE /dist/
COPY plugin.go /dist/
COPY plugin.go /dist/
COPY vendor /dist/

COPY deploy/install.sh /

ENTRYPOINT ["/install.sh"]
