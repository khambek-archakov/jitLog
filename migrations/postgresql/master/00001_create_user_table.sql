-- +goose Up
create table user
(
    id              bigserial primary key,
    telegram_id     bigint      not null unique,
    name            text        not null,
    age             smallint,
    belt            smallint    not null default 0,
    onboarding_step smallint    not null default 0,
    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now()
);

-- +goose Down
drop table user;
