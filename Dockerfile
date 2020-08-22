###
# Builder
FROM golang:alpine as BUILDER

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -o rotating-rsync-backup .

###
# Final image
FROM alpine:latest
LABEL maintainer="William Hefter <wh@elehq.de>"

RUN apk add openssh-client rsync

WORKDIR /app

COPY --from=BUILDER /app/rotating-rsync-backup .

ENTRYPOINT ["./rotating-rsync-backup"]
