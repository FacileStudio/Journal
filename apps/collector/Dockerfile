FROM golang:1.24-alpine AS build

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} go build -o bin/collector .

FROM alpine:3.20

WORKDIR /app

COPY --from=build /src/bin/collector /app/collector

ENTRYPOINT ["/app/collector"]
