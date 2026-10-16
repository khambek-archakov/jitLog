-- +goose Up
alter table "user" add column last_activity_at timestamptz not null default now();
alter table "user" add column winback_sent_at timestamptz;
alter table "user" add column winback_declined_at timestamptz;

-- +goose Down
alter table "user" drop column winback_declined_at;
alter table "user" drop column winback_sent_at;
alter table "user" drop column last_activity_at;
