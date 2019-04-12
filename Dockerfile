FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git ca-certificates


WORKDIR /app
COPY . .

# Build the binary.
RUN CGO_ENABLED=0 go build


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy our static executable.
COPY --from=builder /app/index.html /app/gonator /

# Run the hello binary.
ENTRYPOINT ["/gonator"]
