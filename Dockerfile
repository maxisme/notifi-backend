FROM golang:alpine AS builder
COPY . /app/
WORKDIR /app
RUN go build -o app


FROM alpine
COPY . /app/
COPY --from=builder /app/app /app/app
HEALTHCHECK CMD curl --fail http://localhost:8080/ || exit 1
CMD ["/app/app"]