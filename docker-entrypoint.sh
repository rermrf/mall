#!/bin/sh
set -e

# Replace infrastructure hosts in config files when environment variables are set.
# This allows switching between remote (rermrf.icu) and local (Docker) infrastructure.
#
# Usage in docker-compose:
#   environment:
#     MYSQL_HOST: mysql
#     REDIS_HOST: redis
#     KAFKA_HOST: kafka
#     ETCD_HOST: etcd
#     ES_HOST: elasticsearch

for f in /config/*.yaml; do
  [ -f "$f" ] || continue

  if [ -n "$MYSQL_HOST" ]; then
    sed -i "s/rermrf\.icu:3306/$MYSQL_HOST:3306/g" "$f"
  fi
  if [ -n "$REDIS_HOST" ]; then
    sed -i "s/rermrf\.icu:6379/$REDIS_HOST:6379/g" "$f"
  fi
  if [ -n "$KAFKA_HOST" ]; then
    sed -i "s/rermrf\.icu:9094/$KAFKA_HOST:9094/g" "$f"
  fi
  if [ -n "$ETCD_HOST" ]; then
    sed -i "s/rermrf\.icu:2379/$ETCD_HOST:2379/g" "$f"
  fi
  if [ -n "$ES_HOST" ]; then
    sed -i "s/rermrf\.icu:9200/$ES_HOST:9200/g" "$f"
  fi
done

exec "$@"
