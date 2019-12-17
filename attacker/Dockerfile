FROM tiangolo/uwsgi-nginx-flask:python3.7

RUN apt update
RUN apt install -y nmap net-tools dnsutils

COPY ./app /app
ENTRYPOINT ["python3", "/app/main.py"]
