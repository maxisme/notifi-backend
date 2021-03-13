FROM golang:1.16-alpine AS builder
COPY src /app
WORKDIR /app
RUN go build -o app


FROM alpine
COPY --from=builder /app/app /app
COPY migrations migrations

ARG COMMIT_HASH
ENV COMMIT_HASH=$COMMIT_HASH

CMD ["./app"]