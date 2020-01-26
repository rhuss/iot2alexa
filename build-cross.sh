#!/bin/sh

# Create Buildkit Builder and use it:
#
# docker buildx create --name xbuilder
# docker buildx use xbuilder

docker buildx build --platform linux/arm/v7,linux/amd64 --progress plain --push -t rhuss/iot2alexa:${1:-latest} .
