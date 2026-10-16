-- +goose Up
alter table "user" add column reminders_enabled boolean not null default false;

create table reminder_sent
(
    user_id    bigint      not null references "user" (id),
    week_start date        not null,
    sent_at    timestamptz not null default now(),
    primary key (user_id, week_start)
);

-- +goose Down
drop table reminder_sent;
alter table "user" drop column reminders_enabled;
