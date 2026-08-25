ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-trixie AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
ARG CMD=api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${CMD}

FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /
COPY --from=build /out/app /app
COPY apps/api/migrations /migrations
USER nonroot:nonroot
ENTRYPOINT ["/app"]
