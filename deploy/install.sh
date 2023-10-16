#!/bin/sh
set -e

# Copy this plugin to where Traefik is expecting it
mkdir -p /plugins-local/src/github.com/nscuro/traefik-plugin-geoblock
cp -R /dist/* /plugins-local/src/github.com/nscuro/traefik-plugin-geoblock/
cp /dist/.traefik.yml /plugins-local/src/github.com/nscuro/traefik-plugin-geoblock/
