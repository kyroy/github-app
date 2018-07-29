#!/usr/bin/env bash

#CONFIG=config.yaml
#if [ -f myconfig.yaml ]; then
#    CONFIG=myconfig.yaml
#fi
#        ; echo "using $CONFIG" \

docker build -t github-app . \
    && (
        docker rm -f github-app \
        ; docker run --rm -d -p 8080:8080 \
            -v /var/run/docker.sock:/var/run/docker.sock \
            -v $PWD/kyroy-s-testapp.2018-07-28.private-key.pem:/kyroy-s-testapp.2018-07-28.private-key.pem \
            --name github-app github-app \
        ; docker logs -f github-app
    )
