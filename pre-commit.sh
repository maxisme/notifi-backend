#!/bin/bash

if ! (cd src && golangci-lint run); then
  exit 1
fi

if ! (cd infra && terraform fmt -check); then
  (cd infra && terraform fmt)
  exit 1
fi