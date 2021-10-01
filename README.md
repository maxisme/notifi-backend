<p align="center"><img height="150px" src="https://raw.githubusercontent.com/maxisme/notifi/master/images/bell.png"></p>

# [notifi.it](https://notifi.it/)

## [Mac App](https://github.com/maxisme/notifi) | [Website](https://github.com/maxisme/notifi.it) | Backend

[![Build Status](https://github.com/maxisme/notifi-backend/workflows/notifi/badge.svg)](https://github.com/maxisme/notifi-backend/actions)
[![Coverage Status](https://codecov.io/gh/maxisme/notifi-backend/branch/master/graph/badge.svg)](https://codecov.io/gh/maxisme/notifi-backend)
[![Supported Go Versions](https://img.shields.io/badge/go-1.16-green)](https://github.com/maxisme/notifi-backend/actions)
[![Linter](https://img.shields.io/badge/lint-golangci--lint-blue)](https://golangci-lint.run/)
[![Go Report Card](https://goreportcard.com/badge/github.com/maxisme/notifi-backend)](https://goreportcard.com/report/github.com/maxisme/notifi-backend)


## Run App


## Run linter
Install https://golangci-lint.run/usage/install/#local-installation
```bash
bash pre-commit.sh
```

## Add pre-commit hook

```bash
ln -s $(pwd)/pre-commit.sh $(pwd)/.git/hooks/pre-commit
chmod +x $(pwd)/.git/hooks/pre-commit
```

# tokens
## AWS AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
 - https://console.aws.amazon.com
 - Security Credentials (under username)
 - Access keys (access key ID and secret access key)



# Worker
test locally by installing:
```bash
npm install -g miniflare
```

Then running:
```bash
miniflare worker/github-release.js -w -d
```
