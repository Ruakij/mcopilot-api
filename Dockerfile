FROM ghcr.io/go-rod/rod as base

# Install novnc and dependencies
RUN apt update && apt install -y --no-install-recommends xvfb x11vnc fluxbox git python3 && \
    rm -rf /var/lib/apt/lists/*
RUN git clone https://github.com/novnc/noVNC.git /opt/novnc && \
    git clone https://github.com/novnc/websockify.git /opt/novnc/utils/websockify

COPY entrypoint.sh /entrypoint.sh

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

ENTRYPOINT ["./entrypoint.sh"]  

EXPOSE 5000
EXPOSE 6080
