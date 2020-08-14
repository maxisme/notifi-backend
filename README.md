<p align="center"><img height="150px" src="https://github.com/maxisme/notifi/raw/master/notifi/images/bell.png"></p>

# [notifi.it](https://notifi.it/)

## [Mac App](https://github.com/maxisme/notifi) | [Website](https://github.com/maxisme/notifi.it) | Backend

[![Build Status](https://github.com/maxisme/notifi-backend/workflows/notifi/badge.svg)](https://github.com/maxisme/notifi-backend/actions)
[![Coverage Status](https://codecov.io/gh/maxisme/notifi-backend/branch/master/graph/badge.svg)](https://codecov.io/gh/maxisme/notifi-backend)
[![Supported Go Versions](https://img.shields.io/badge/go-1.12%20|%201.13%20|%201.14-green&style=plastic)](https://github.com/maxisme/notifi-backend/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/maxisme/notifi-backend)](https://goreportcard.com/report/github.com/maxisme/notifi-backend)

Add `.env` to project:
```
SERVER_KEY=
ENCRYPTION_KEY=
sentry_dsn=
```

To create new migrations run:
```
$ migrate create -ext sql -dir sql/ -seq "description"
```


## Running locally
For testing simply run:
```bash
$ docker-compose up -d db
$ docker-compose up migrate
$ docker-compose up -d redis
$ go run .
```