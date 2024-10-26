#!/bin/bash
IMAGE_PROJECT=$1
IMAGE_NAME=brew-api
VERSION=$2

# Make full image name
IMAGE=${IMAGE_PROJECT}/${IMAGE_NAME}:${VERSION}

docker build -t ${IMAGE} --platform linux/amd64 . 

docker push ${IMAGE_PROJECT}/${IMAGE_NAME}:${VERSION}
