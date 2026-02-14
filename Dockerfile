FROM golang:1.25-alpine AS builder

# Install git (if you use private modules)
RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source (required one, and in order of change frequency)
COPY config ./config
COPY pkg ./pkg
COPY internals ./internals
COPY cmd ./cmd

# Make a build directory
RUN mkdir build

# Build binary (static)  and put binary in build folder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/build/app ./cmd/api/main.go   

# ---------- Runtime Stage ----------
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/build/app ./app

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["./app"]