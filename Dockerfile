FROM ubuntu:trusty

LABEL description="hardforkdemo"
LABEL version="1.0"
LABEL maintainer "holdstockjamie@gmail.com"

USER root
WORKDIR /root

COPY docker_context/public            /root/public
COPY docker_context/hardforkdemo      /root/hardforkdemo

CMD ./hardforkdemo
