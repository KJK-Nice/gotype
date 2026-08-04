# Railway Phoenixd: root for volume perms + IPv6 private network bind.
# Official image runs as uid 1000 and fails writing seed on Railway volumes.
FROM acinq/phoenixd:latest
USER root
ENV PHOENIX_DATADIR=/phoenix/.phoenix
ENTRYPOINT []
# PHOENIXD_WEBHOOK_SECRET must match gotype-ssh PHOENIXD_WEBHOOK_SECRET for HMAC verify.
CMD ["/bin/bash", "-c", "set -euo pipefail; args=(--agree-to-terms-of-service --http-bind-ip :: --http-password \"${PHOENIXD_API_PASSWORD}\"); if [[ -n \"${PHOENIXD_WEBHOOK_SECRET:-}\" ]]; then args+=(--webhook-secret \"${PHOENIXD_WEBHOOK_SECRET}\"); fi; exec /phoenix/phoenixd \"${args[@]}\""]
