#!/usr/bin/bash

docker build -t harbor.jdonszelmann.nl/library/short .
docker push harbor.jdonszelmann.nl/library/short
