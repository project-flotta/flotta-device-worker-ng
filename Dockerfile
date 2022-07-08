FROM golang:1.18 AS build

WORKDIR /app

COPY . .
RUN if [ ! -d "./vendor" ]; then make build.vendor; fi

RUN GOOS=linux GOARCH=amd64 make build

################
#   Run step   #
################
FROM gcr.io/distroless/base

ENTRYPOINT ["/app/bin/device-worker"]
