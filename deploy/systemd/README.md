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

## Импорт из файла

Если `ajptour.com` отвечает странице списка Cloudflare-челленджем именно
с IP сервера (`cf-mitigated: challenge` в заголовках — частое дело для
datacenter-адресов, вроде AWS) — обходить это кодом нельзя, но можно
прочитать уже сохранённую человеком копию страницы:

1. На своей машине откройте в обычном браузере
   `https://ajptour.com/en/federation/1/events`.
2. Сохраните страницу как **«Веб-страница, только HTML»**
   (не «полностью» — лишние ассеты не нужны, разбор смотрит только на
   `data-*`-атрибуты карточек, которые одинаковы что в HTML с сервера,
   что в сохранённом браузером файле).
3. Скопируйте файл на сервер:
   ```sh
   scp page.html ubuntu@<сервер>:/tmp/page.html
   ```
4. На сервере — рабочая папка и права важны: бинарник читает `.env` и
   `configs/` из своей директории, а `.env` доступен только root:
   ```sh
   cd /opt/jitlog/import-ajp
   sudo ./ajp-import --from-file /tmp/page.html --dry-run   # сначала посмотреть
   sudo ./ajp-import --from-file /tmp/page.html             # затем по-настоящему
   ```

Особенности этого режима:
- **Ни одного сетевого запроса** — ни к сайту, ни к `robots.txt`.
- Если файл не читается, пустой, без `section#upcoming-events` или ни
  одной карточки — выход с ненулевым кодом и понятным сообщением в
  **stderr**; в Telegram про это не пишет (вы и так видите ошибку в
  терминале) — остальные сообщения (новые соревнования, расхождения)
  по-прежнему уходят в Telegram как обычно.
- Если файлу больше 3 суток — предупреждение в stderr, но разбор всё
  равно продолжается.
- В сообщении о новых соревнованиях добавляется строка
  «Источник: файл page.html», чтобы было видно, что это не живые данные
  с сайта.
- Повторный запуск на том же файле не создаёт дублей — дедуп идёт по
  нормализованному URL, как и при обычном запуске по сети.
