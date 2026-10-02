#!/bin/bash
set -e
mkdir -p files
go build -o awadori .
./awadori "$@"
