-- +goose Up
create table schedule_slot
(
    id              bigserial primary key,
    user_id         bigint      not null references "user" (id),
    day_of_week     smallint    not null,
    time_minutes    smallint    not null,
    training_type   smallint    not null,
    created_at      timestamptz not null default now()
);

create index schedule_slot_user_id_idx on schedule_slot (user_id);

create table schedule_draft
(
    id              bigserial primary key,
    user_id         bigint      not null unique references "user" (id),
    step            smallint    not null default 0,
    day_of_week     smallint,
    time_minutes    smallint,
    training_type   smallint    not null default 0,
    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now()
);

-- +goose Down
drop table schedule_draft;
drop table schedule_slot;
