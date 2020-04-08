FROM python:3-alpine
ARG version=dirty

WORKDIR /usr/src/app

RUN addgroup -S idsgroup
RUN adduser -S -D 999 idsgroup

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY install.py ./
COPY data ./data
COPY config.yaml ./
RUN echo $version > ./version.txt

USER 999

ENTRYPOINT ["python", "install.py"]
