# API image: compiles the Go server and runs it as an unprivileged user.
ARG GO_IMAGE=golang:1.27.4-alpine
FROM ${GO_IMAGE} AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.20

RUN adduser -D -u 10001 app \
 && mkdir -p /app/uploads \
 && chown app /app/uploads

WORKDIR /app
COPY --from=build /out/api /app/api
COPY data /app/data

USER app
EXPOSE 8080

HEALTHCHECK --interval=5s --timeout=3s --retries=20 \
  CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

CMD ["/app/api"]
