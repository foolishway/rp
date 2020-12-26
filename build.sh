#!/bin/bash

GOOS=darwin go build -o rp *.go 
cp rp $GOPATH/bin
rm ./rp
