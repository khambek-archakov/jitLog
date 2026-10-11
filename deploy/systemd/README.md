# AJP competition import — systemd

Runs `cmd/cron/competition_import/ajp` once a night via a `oneshot`
service triggered by a `timer`. The bot process itself never schedules
this — systemd does.

## Install

```sh
# 1. Build the binary
sudo mkdir -p /opt/jitlog/import-ajp
go build -o /opt/jitlog/import-ajp/ajp-import ./cmd/cron/competition_import/ajp

# 2. Ship the city dictionary alongside it (AJP_CITY_DICTIONARY_PATH
#    defaults to configs/ajp_cities.txt, relative to WorkingDirectory)
sudo cp -r configs /opt/jitlog/import-ajp/configs

# 3. Env file — reuses the same variable names as the main bot's own
#    config (see cmd/service/main.go): TELEGRAM_TOKEN, DATABASE_URL,
#    ADMIN_TELEGRAM_ID. AJP_COUNTRIES and AJP_CITY_DICTIONARY_PATH are
#    optional — leave them out to get the defaults (RU,BY,AZ,AM,KZ and
#    configs/ajp_cities.txt).
sudo tee /opt/jitlog/import-ajp/.env <<'EOF'
TELEGRAM_TOKEN=
DATABASE_URL=
ADMIN_TELEGRAM_ID=
# AJP_COUNTRIES=RU,BY,AZ,AM,KZ
# AJP_CITY_DICTIONARY_PATH=configs/ajp_cities.txt
EOF
sudo chmod 600 /opt/jitlog/import-ajp/.env

# 4. Install the units
sudo cp deploy/systemd/jitlog-import-ajp.service \
        deploy/systemd/jitlog-import-ajp.timer \
        deploy/systemd/jitlog-import-ajp-notify-failure@.service \
        /etc/systemd/system/

sudo systemctl daemon-reload
sudo systemctl enable --now jitlog-import-ajp.timer
```

## Manual run

```sh
# Real run, right now, outside the timer's own schedule:
sudo systemctl start jitlog-import-ajp.service

# Preview only — parses and compares, writes nothing, sends nothing:
sudo -u jitlog /opt/jitlog/import-ajp/ajp-import --dry-run
```

(Run `--dry-run` as whatever user owns `/opt/jitlog/import-ajp/.env` —
root works too if that's what owns it; it never touches systemd itself.)

## Logs

```sh
journalctl -u jitlog-import-ajp.service -n 100
journalctl -u jitlog-import-ajp.service --since today
journalctl -u jitlog-import-ajp.timer    # when it last ran / next runs
```

A run that finds another one already in progress (the `pg_try_advisory_lock`
guard) exits 0 and logs "another AJP import run is already in progress" —
that's expected if a previous run is still finishing, not a failure.

## Failure notification

`jitlog-import-ajp.service` declares `OnFailure=` pointing at the
`jitlog-import-ajp-notify-failure@.service` template, so systemd itself
starts it automatically on any non-zero exit or timeout — a direct
`curl` call to the Bot API using the same `.env`, no extra binary.
