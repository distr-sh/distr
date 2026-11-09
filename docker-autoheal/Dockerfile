FROM alpine:3.24.4

# Set to the current date in CI, so BuildKit reruns the upgrade daily instead of reusing a cached
# layer with packages from before the latest Alpine security fixes.
ARG APK_UPGRADE_DATE
RUN apk upgrade --no-cache && apk add --no-cache curl jq

ENV AUTOHEAL_CONTAINER_LABEL=autoheal \
  AUTOHEAL_START_PERIOD=0 \
  AUTOHEAL_INTERVAL=5 \
  AUTOHEAL_DEFAULT_STOP_TIMEOUT=10 \
  DOCKER_SOCK=/var/run/docker.sock \
  CURL_TIMEOUT=30 \
  WEBHOOK_URL="" \
  WEBHOOK_JSON_KEY="content" \
  APPRISE_URL="" \
  POST_RESTART_SCRIPT=""

COPY --chmod=755 docker-entrypoint /

HEALTHCHECK --interval=5s CMD pgrep -f autoheal || exit 1

ENTRYPOINT ["/docker-entrypoint"]

CMD ["autoheal"]
