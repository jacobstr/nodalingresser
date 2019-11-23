FROM golang:1.13-alpine3.10 AS build
WORKDIR /app
COPY . /app
RUN go build -o /nodalingresser ./cmd/nodalingresser

FROM alpine:3.10
COPY --from=build /nodalingresser /nodalingresser
