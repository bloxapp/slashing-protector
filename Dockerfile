ARG  BUILDER_IMAGE=golang:1.17.4-alpine
############################
# STEP 1 build executable binary
############################
FROM ${BUILDER_IMAGE} as builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata gcc g++ && update-ca-certificates

# Create an unprivileged user.
ENV USER=protector
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -o /app/slashing-protector -a \
    -ldflags '-linkmode external -extldflags "-static -lm"' \
    ./cmd/slashing-protector

############################
# STEP 2 build a small image
############################
FROM alpine:3.15

# Import from builder.
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy our static executable
COPY --from=builder /app/slashing-protector /app/slashing-protector

# Use an unprivileged user.
USER protector:protector

# Run the binary.
ENTRYPOINT ["/app/slashing-protector"]