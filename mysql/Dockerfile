FROM mysql:9.3

HEALTHCHECK --interval=5s --timeout=3s --retries=3 \
    CMD mysqladmin ping -h localhost || exit 1
