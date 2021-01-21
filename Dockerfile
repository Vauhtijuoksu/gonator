FROM golang:1.15.7-alpine3.12 AS builder

RUN apk update && apk add --no-cache git ca-certificates


WORKDIR /app
COPY . .

# Build the binary.
RUN CGO_ENABLED=0 go build


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy our static executable.
COPY --from=builder /app/static /static
COPY --from=builder /app/gonator /

# Run the hello binary.
ENTRYPOINT ["/gonator"]
