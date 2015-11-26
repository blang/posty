FROM ubuntu:trusty
MAINTAINER Benedikt Lang <mail@blang.io>
RUN apt-get update -yq
RUN apt-get install ca-certificates -yq

COPY ./posty /posty
COPY ./frontend/dist /public
EXPOSE 8080
CMD ["/posty","-frontend-path","/public"]

