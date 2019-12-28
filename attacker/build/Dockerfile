FROM debian:latest

RUN apt update
RUN apt install -y nmap net-tools dnsutils

COPY ./scan.sh /scan.sh
ENTRYPOINT ["bash", "-c" ,"/scan.sh"]
