#!/bin/bash
if ! (cd src && golangci-lint run); then
  exit 1
fi

if ! (cd infra && terraform fmt -recursive -check); then
  (cd infra && terraform fmt -recursive)
  exit 1
fi