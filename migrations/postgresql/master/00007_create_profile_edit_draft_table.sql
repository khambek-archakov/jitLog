-- +goose Up
create table profile_edit_draft
(
    user_id    bigint primary key references "user" (id),
    created_at timestamptz not null default now()
);

-- +goose Down
drop table profile_edit_draft;
