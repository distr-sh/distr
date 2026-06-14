FROM alpine:3.24

COPY crontab /etc/crontabs/root

CMD ["busybox", "crond", "-f", "-L", "/dev/stdout"]
