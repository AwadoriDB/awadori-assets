#!/bin/bash
set -e
mkdir -p files
go build -o awano-winter .
./awano-winter "$@"
