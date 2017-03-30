FROM alpine:3.4
RUN apk add --no-cache \
    python \
    netcat-openbsd
COPY udpping.sh tcpping.sh tcppingasym.sh responder.py /code/
WORKDIR /code/
RUN chmod +x udpping.sh && chmod +x tcpping.sh && chmod +x tcppingasym.sh
CMD ["python", "responder.py"]
