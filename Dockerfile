FROM oven/bun:1 AS client-build
WORKDIR /client
COPY apps/client/package.json apps/client/bun.lock* ./
RUN bun install --frozen-lockfile
COPY apps/client/ .
ARG BUILD_TIMESTAMP
RUN echo "Building at ${BUILD_TIMESTAMP:-unknown}" && bun run build

FROM golang:1.26-alpine AS api-build

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /repo/apps/api

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api ./

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o bin/api .

# AVATAR_DIR has to exist in the image, owned by the user the API runs as.
# Docker seeds a fresh named volume from the image path it is mounted over,
# including its ownership — so a path that does not exist here produces a
# root-owned volume, and an API running as nonroot cannot write an avatar into
# it. The failure is a permission denied deep in the OIDC callback, logged as a
# warning and swallowed, which is a long way from the cause.
RUN mkdir -p /data/avatars

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=api-build /repo/apps/api/bin/api /api
COPY --from=client-build /client/build /client
COPY --from=api-build --chown=nonroot:nonroot /data /data

# The distroless base can carry its own WorkingDir (/home/nonroot on the
# :nonroot variant), which would make a relative ./client resolve there and
# the SPA silently not be served at all. Be explicit.
ENV CLIENT_DIR=/client

EXPOSE 4010

USER nonroot:nonroot

ENTRYPOINT ["/api"]
