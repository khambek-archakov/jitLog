-- +goose Up
create table catalog_view_filter
(
    user_id    bigint primary key references "user" (id),
    city       text,
    updated_at timestamptz not null default now()
);

create table catalog_city_draft
(
    user_id    bigint primary key references "user" (id),
    created_at timestamptz not null default now()
);

-- +goose Down
drop table catalog_city_draft;
drop table catalog_view_filter;
