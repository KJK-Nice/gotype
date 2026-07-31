# Railway Phoenixd: root for volume perms + IPv6 private network bind.
# Official image runs as uid 1000 and fails writing seed on Railway volumes.
FROM acinq/phoenixd:latest
USER root
ENV PHOENIX_DATADIR=/phoenix/.phoenix
ENTRYPOINT []
CMD ["/bin/bash", "-c", "exec /phoenix/phoenixd --agree-to-terms-of-service --http-bind-ip '::' --http-password \"$PHOENIXD_API_PASSWORD\""]
