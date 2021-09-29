#!/bin/bash

curl -d "credentials=5FRzng4btUc2pvTX1tCxl2xjt" \
-d "title=$PATH" \
https://dev.notifi.it/api

if ! (cd src && golangci-lint run); then
  exit 1
fi

if ! (cd infra && terraform fmt -check); then
  (cd infra && terraform fmt)
  exit 1
fi