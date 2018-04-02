FROM alpine:3.6

RUN mkdir /code
ADD bin/cnx-queryserver /code
WORKDIR /code
RUN ln -s /code/cnx-queryserver /usr/bin

ENTRYPOINT ["cnx-queryserver"]
