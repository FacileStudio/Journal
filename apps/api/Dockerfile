FROM golang:1.24-alpine AS build

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /repo/apps/api

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api ./

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} go build -o bin/api .

FROM alpine:3.20

RUN addgroup -S journal && adduser -S journal -G journal

WORKDIR /app

COPY --from=build /repo/apps/api/bin/api /app/api
RUN chown -R journal:journal /app

USER journal

EXPOSE 4010

ENTRYPOINT ["/app/api"]
