FROM python:3

ARG UID=1000

RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt/lists \
    apt-get update

RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt/lists \
    apt-get -y upgrade

RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt/lists \
    apt-get -y install rpm

RUN --mount=type=bind,source=requirements.txt,target=/root/requirements.txt\
    pip install -r /root/requirements.txt
