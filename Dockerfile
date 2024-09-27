FROM golang:1.22.1-alpine3.19 as build
WORKDIR /core-api
COPY . /core-api
RUN apk add make bash which && make go-build

FROM docker.io/library/ubuntu:latest
COPY --from=build /core-api/cmd/core-api-server/core-api-server /usr/bin/
EXPOSE 80
EXPOSE 443
WORKDIR /usr/bin