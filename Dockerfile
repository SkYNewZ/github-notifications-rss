FROM golang:1.19-alpine as builder
WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY config.go config.go
COPY feed.go feed.go
COPY main.go main.go

RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -a -o /github-notifications-rss .

FROM gcr.io/distroless/static:nonroot
WORKDIR /

ARG BUILD_DATE
ARG VCS_REF

LABEL maintainer="Quentin Lemaire <quentin@lemairepro.fr>"
LABEL org.label-schema.schema-version="1.0"
LABEL org.label-schema.build-date=$BUILD_DATE
LABEL org.label-schema.name="github-notifications-rss"
LABEL org.label-schema.description="Generate a JSON feed from your Github notifications"
LABEL org.label-schema.url="https://github.com/SkYNewZ/github-notifications-rss"
LABEL org.label-schema.vcs-ref=$VCS_REF

COPY --from=builder /github-notifications-rss /github-notifications-rss
USER 65532:65532

ENTRYPOINT ["/github-notifications-rss"]
