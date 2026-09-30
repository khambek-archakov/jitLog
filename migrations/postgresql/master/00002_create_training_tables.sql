-- +goose Up
create table training
(
    id                bigserial primary key,
    user_id           bigint      not null references "user" (id),
    training_date     date        not null,
    training_type     smallint    not null,
    duration_minutes  integer     not null,
    notes             text,
    created_at        timestamptz not null default now()
);

create index training_user_id_idx on training (user_id);

create table training_draft
(
    id                bigserial primary key,
    user_id           bigint      not null unique references "user" (id),
    step              smallint    not null default 0,
    training_date     date,
    training_type     smallint    not null default 0,
    duration_minutes  integer,
    notes             text,
    created_at        timestamptz not null default now(),
    updated_at        timestamptz not null default now()
);

-- +goose Down
drop table training_draft;
drop table training;
