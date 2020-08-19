FROM golang:alpine AS builder
COPY . /app/
WORKDIR /app
RUN go build -o app


FROM alpine

ARG COMMIT_HASH
ENV COMMIT_HASH=$COMMIT_HASH

WORKDIR /app
COPY --from=builder /app/app /app/app
COPY web /app/web
RUN apk add curl
HEALTHCHECK CMD curl --fail http://localhost:8080/health || exit 1
CMD ["./app"]