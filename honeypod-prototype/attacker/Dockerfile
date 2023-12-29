FROM debian:latest

RUN apt update && apt install -y \
    curl \
    dnsutils \
    net-tools \
    nmap

COPY scan.sh /scan.sh

ENTRYPOINT ["bash", "-c" ,"/scan.sh"]
