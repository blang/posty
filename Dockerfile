FROM ubuntu:trusty
MAINTAINER Benedikt Lang <mail@blang.io>
RUN apt-get update -yq
RUN apt-get install ca-certificates -yq

COPY ./posty /posty
COPY ./frontend/dist /public
COPY ./ebsrun.sh /ebsrun.sh
EXPOSE 8080
CMD ["/ebsrun.sh"]

