FROM alpine:3.6

RUN mkdir /code
RUN chgrp -R 0 /code && \
    chmod -R g=u /code
ADD bin/cnx-queryserver /code
WORKDIR /code
RUN ln -s /code/cnx-queryserver /usr/bin

ENTRYPOINT ["cnx-queryserver"]
