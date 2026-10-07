-- +goose Up
create table user_competition
(
    id             bigserial primary key,
    user_id        bigint      not null references "user" (id),
    competition_id bigint references competition (id),
    title          text        not null,
    date           date        not null,
    end_date       date,
    city           text,
    url            text,
    result         text,
    created_at     timestamptz not null default now(),
    updated_at     timestamptz not null default now(),
    check (end_date is null or end_date >= date)
);

create index idx_user_competition_user_date on user_competition (user_id, date);

create table user_competition_draft
(
    id         bigserial primary key,
    user_id    bigint      not null unique references "user" (id),
    step       smallint    not null default 0,
    title      text,
    date       date,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table user_competition_edit_draft
(
    user_id           bigint      primary key references "user" (id),
    user_competition_id bigint    not null references user_competition (id) on delete cascade,
    field             smallint    not null,
    created_at        timestamptz not null default now()
);

-- +goose Down
drop table user_competition_edit_draft;
drop table user_competition_draft;
drop table user_competition;
