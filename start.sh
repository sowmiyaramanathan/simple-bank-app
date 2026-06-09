#!/bin/sh
set -e

if [ -f /app/app.env ]; then
	set -a
	. /app/app.env
	set +a
fi

if [ -z "$DB_SOURCE" ]; then
	echo "DB_SOURCE is empty. Set it with docker run -e DB_SOURCE=... or provide /app/app.env in the image."
	exit 1
fi

echo "run db migration"
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose up

echo "start the app"
exec "$@"