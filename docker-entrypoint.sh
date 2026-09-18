#!/bin/sh
# BirthDayMemo container entrypoint.
#
# The app keeps config.json, birthdaymemo.db, logs/ and languages/ next to
# its executable, so it must run from the persistent /data directory.
# The pristine binary lives in the read-only image at
# /opt/birthdaymemo/birthdaymemo and is (re)copied into /data on every
# start. This keeps user data persistent across container recreations
# while guaranteeing that rebuilt images actually run the new binary
# (a plain volume mount over the binary would shadow updates).
set -e

cp -f /opt/birthdaymemo/birthdaymemo /data/birthdaymemo

# Extra args are forwarded, e.g.:
#   docker compose run --rm birthdaymemo -reset-admin-password
# Configuration comes from BIRTHDAYMEMO_* environment variables
# (see compose.yaml); no interactive setup is performed here.
exec /data/birthdaymemo "$@"
