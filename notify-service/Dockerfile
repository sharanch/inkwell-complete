FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
ENV GONOSUMDB=* GOPROXY=direct GOFLAGS=-mod=mod
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /service ./cmd/server

FROM scratch
COPY --from=builder /service /service
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8084
ENTRYPOINT ["/service"]
