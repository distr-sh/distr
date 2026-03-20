FROM alpine:3.23

COPY crontab /etc/crontabs/root

CMD ["busybox", "crond", "-f", "-L", "/dev/stdout"]
