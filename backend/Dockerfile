# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS base
WORKDIR /app
RUN apk add --no-cache git ca-certificates

FROM base AS dev
RUN go install github.com/air-verse/air@latest
COPY go.mod ./
RUN go mod download || true
COPY . .
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

FROM base AS build
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20 AS prod
RUN apk add --no-cache ca-certificates wget
WORKDIR /
COPY --from=build /server /server
EXPOSE 8080
USER nobody
ENTRYPOINT ["/server"]
