<p align="center"><img height="150px" src="https://github.com/maxisme/notifi/raw/master/notifi/images/bell.png"></p>

# [notifi.it](https://notifi.it/)

## [Mac App](https://github.com/maxisme/notifi) | [Website](https://github.com/maxisme/notifi.it) | Backend

[![Build Status](https://github.com/maxisme/notifi-backend/workflows/notifi/badge.svg)](https://github.com/maxisme/notifi-backend/actions)
[![Supported Go Versions](https://img.shields.io/badge/Go%20Versions-1.12%2C%201.13%2C%201.14-green&style=plastic)](https://github.com/maxisme/notifi-backend/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/maxisme/notifi-backend)](https://goreportcard.com/report/github.com/maxisme/notifi-backend)
[![Coverage Status](https://codecov.io/gh/maxisme/notifi-backend/branch/master/graph/badge.svg)](https://codecov.io/gh/maxisme/notifi-backend)

Add `.env` to project:
```
server_key=
encryption_key=
sentry_dsn=
```

To install migrate:
```
$ curl -L https://packagecloud.io/mattes/migrate/gpgkey | apt-key add -
$ echo "deb https://packagecloud.io/mattes/migrate/ubuntu/ xenial main" > /etc/apt/sources.list.d/migrate.list
$ apt-get update
$ apt-get install -y migrate
```

To initialise schema first create a database `notifi` then run:
```
migrate -database mysql://root:@/notifi up
```

To create new migrations run:
```
$ migrate create -ext sql -dir sql/ -seq remove_col
```
