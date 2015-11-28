FROM golang:1.5
MAINTAINER Benedikt Lang <mail@blang.io>

RUN go get -v github.com/skelterjohn/wgo
ENV HOME /home
COPY ./fakegitconfig /home/.gitconfig
WORKDIR /data
EXPOSE 6060
VOLUME ["/data"]
