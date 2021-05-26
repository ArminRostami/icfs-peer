# build stage
FROM golang:alpine AS builder

RUN apk add --no-cache git

WORKDIR /app/src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a ./cmd/main.go


# final stage
FROM ubuntu:bionic

RUN apt-get update && \
    apt-get --no-install-recommends --yes install \
    chromium-browser \
    dumb-init \
    fontconfig \
    && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd chromium && \
    useradd --create-home --gid chromium chromium && \
    chown --recursive chromium:chromium /home/chromium/

VOLUME ["/home/chromium/.fonts"]

COPY --chown=chromium:chromium entrypoint.sh /home/chromium/

WORKDIR /app/bin
COPY --from=builder /app/src/main .

USER chromium

EXPOSE 9222

CMD ["dumb-init", "--", "/bin/sh", "/home/chromium/entrypoint.sh"]

