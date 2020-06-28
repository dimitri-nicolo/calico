FROM docker.elastic.co/elasticsearch/elasticsearch:7.3.2

RUN yum -y update && yum -y upgrade
RUN yum clean all

COPY cleanup.sh /
RUN /cleanup.sh

