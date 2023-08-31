#!/bin/sh
cd frontend
npm run build
cd ..
go-bindata -fs -pkg "http_api" -prefix "frontend/dist/" frontend/dist/ frontend/dist/assets/
