FROM docker.elastic.co/elasticsearch/elasticsearch:7.6.2

RUN yum -y update && yum -y upgrade && yum clean all

COPY cleanup.sh /
RUN /cleanup.sh

