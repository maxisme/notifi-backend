FROM golang:1.12-alpine

ADD . /app/
WORKDIR /app
RUN go build -o notifi .
CMD ["/app/notifi"]