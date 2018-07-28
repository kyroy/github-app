#!/usr/bin/env bash

#CONFIG=config.yaml
#if [ -f myconfig.yaml ]; then
#    CONFIG=myconfig.yaml
#fi
#        ; echo "using $CONFIG" \

docker build -t github-app . \
    && (
        docker rm -f aws-monitor \
        ; docker run --rm -d -p 8080:8080 -v $PWD:/ --name github-app github-app \
        ; docker logs -f github-app
    )
