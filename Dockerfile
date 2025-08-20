FROM ubuntu:24.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       ca-certificates \
       build-essential \
       git \
       golang-go \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

# Copy Go module files first (go.sum may be absent in this repo)
COPY go.mod ./
RUN if [ -f go.mod ]; then go mod download || true; fi

# Copy remaining source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /kensho/server ./cmd/api

FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /kensho/server /app/server
# Copy configuration files required at runtime
COPY --from=builder /src/configs /app/configs

EXPOSE 8080

ENV PORT=8080

ENTRYPOINT ["/app/server"]
