-- +goose Up
create table competition
(
    id         bigserial primary key,
    title      text        not null,
    date       date        not null,
    end_date   date,
    city       text,
    status     smallint    not null,
    created_by bigint references "user" (id),
    created_at timestamptz not null default now(),
    check (end_date is null or end_date >= date)
);

create index idx_competition_status_date on competition (status, date);

create table competition_source
(
    competition_id bigint  not null references competition (id),
    url            text    not null,
    is_primary     boolean not null default false,
    unique (url)
);

create index idx_competition_source_competition on competition_source (competition_id);

create unique index uq_competition_primary_source
    on competition_source (competition_id) where is_primary;

-- +goose Down
drop table competition_source;
drop table competition;
