#!/usr/bin/env bash

while true;
do {
    echo -e "HTTP/1.1 200 OK\r\n$(date)\r\n\r\n<h1>hello world from $(hostname) on $(date) with port ${PORT}</h1>" |  nc -vl $PORT;

}
done