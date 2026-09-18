#!/bin/sh
# BirthDayMemo container entrypoint.
#
# The app keeps config.json, birthdaymemo.db, logs/ and languages/ next to
# its executable, so it must run from the persisted work directory. In
# compose.yaml that directory is bind-mounted at /opt/birthdaymemo.
#
# The pristine binary lives OUTSIDE the mount in the read-only image at
# /usr/local/bin/birthdaymemo (a plain volume mount over /opt/birthdaymemo
# would shadow a binary baked into that path) and is (re)copied to
# /opt/birthdaymemo/birthdaymemo on every start. This keeps user data
# persistent across container recreations while guaranteeing that rebuilt
# images actually run the new binary.
set -e

mkdir -p /opt/birthdaymemo
cp -f /usr/local/bin/birthdaymemo /opt/birthdaymemo/birthdaymemo
chmod +x /opt/birthdaymemo/birthdaymemo

# Run from the persisted work directory (config/db/logs live next to the exe).
cd /opt/birthdaymemo

# Extra args are forwarded, e.g.:
#   docker compose run --rm birthdaymemo -reset-admin-password
#   (--guest-only from compose.yaml)
exec /opt/birthdaymemo/birthdaymemo "$@"