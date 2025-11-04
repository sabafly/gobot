FROM mariadb:latest

ENV MARIADB_MYSQL_LOCALHOST_USER=1

HEALTHCHECK --interval=5s --timeout=3s --retries=3 \
    CMD mariadb-admin ping -h localhost -p$MYSQL_ROOT_PASSWORD || exit 1
