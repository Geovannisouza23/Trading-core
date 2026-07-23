# syntax=docker/dockerfile:1

# ---- build stage -----------------------------------------------------
FROM golang:1.24-alpine AS build

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# -buildvcs=false: see Makefile comment — this repo may be built from a
# tree nested under an unrelated/incomplete git repo.
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---- runtime stage -----------------------------------------------------
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S trading && adduser -S trading -G trading

COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/worker /usr/local/bin/worker
COPY --from=build /out/migrate /usr/local/bin/migrate

USER trading

EXPOSE 8080

# docker-compose overrides `command` per service (api | worker | migrate).
ENTRYPOINT []
CMD ["/usr/local/bin/api"]
