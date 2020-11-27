#!/bin/sh
# wait-for-postgres.sh
se -e

host="$1"
shift
cmd="$@"

until ./rabtap info --api="$host" --stats; do
  >&2 echo "RabbitMQ is unavailable - sleeping"
  sleep 1
done

>&2 echo "RabbitMQ is up - executing command"
exec $cmd