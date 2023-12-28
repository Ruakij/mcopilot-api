FROM ghcr.io/go-rod/rod as base

# ---- Build ----
FROM golang:1.21-alpine AS build
WORKDIR /build
# Copy sources
ADD . .
# Get dependencies
RUN go get ./cmd/
# Compile
RUN CGO_ENABLED=0 go build -a -o app ./cmd/

# ---- Release ----
FROM base AS release
# Copy build-target
COPY --from=build /build/app .

CMD ["./app"]  

EXPOSE 5000
