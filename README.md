<p align="center"><img height="150px" src="https://github.com/maxisme/notifi/raw/master/notifi/images/bell.png"></p>

# [notifi.it](https://notifi.it/)

## [Mac App](https://github.com/maxisme/notifi) | [Website](https://github.com/maxisme/notifi.it) | Backend

[![Build Status](https://github.com/maxisme/notifi-backend/workflows/notifi/badge.svg)](https://github.com/maxisme/notifi-backend/actions)
[![Coverage Status](https://codecov.io/gh/maxisme/notifi-backend/branch/master/graph/badge.svg)](https://codecov.io/gh/maxisme/notifi-backend)
[![Supported Go Versions](https://img.shields.io/badge/go-1.16-green&style=plastic)](https://github.com/maxisme/notifi-backend/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/maxisme/notifi-backend)](https://goreportcard.com/report/github.com/maxisme/notifi-backend)



## Run tests
```bash
$ docker-compose up test
```

## Local development

### Startup dbs
```bash
$ docker-compose up redis db
```
### env variables
```bash
REDIS_HOST=127.0.0.1:6379
DATABASE_HOST=127.0.0.1
DATABASE_USER=notifi
DATABASE_PASS=notifi
DATABASE_NAME=notifi
DATABASE_SSL_DISABLE=1
SERVER_KEY=u2J7b7xA8MndeNS
ENCRYPTION_KEY=6bO9OFNEsqdz3Bl16bO9OFNEsqdz3Bl1
```