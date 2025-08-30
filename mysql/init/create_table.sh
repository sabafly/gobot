#!/usr/bin/env bash
# This script is used to create the gobot_beta database and user in MySQL.
mariadb -uroot -p${MYSQL_ROOT_PASSWORD} <<EOF
CREATE DATABASE IF NOT EXISTS gobot_beta;
GRANT ALL ON gobot_beta.* TO '$MYSQL_USER'@'%';
EOF
