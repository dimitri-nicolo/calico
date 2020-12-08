FROM docker.elastic.co/elasticsearch/elasticsearch:7.10.1

RUN yum -y update && yum -y upgrade && yum clean all

COPY cleanup.sh /
RUN /cleanup.sh

COPY /bin/readiness-probe /
