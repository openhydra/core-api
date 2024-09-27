FROM docker.io/library/ubuntu:latest

ARG APP
COPY cmd/core-api-server/core-api-server /usr/bin/
COPY asserts /doc
EXPOSE 80
EXPOSE 443
WORKDIR /usr/bin