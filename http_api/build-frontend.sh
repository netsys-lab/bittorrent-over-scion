#!/bin/sh
go-bindata -fs -pkg "http_api" -prefix "frontend/" frontend/
