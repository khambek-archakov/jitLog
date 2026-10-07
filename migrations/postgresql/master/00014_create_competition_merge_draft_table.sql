-- +goose Up
create table competition_merge_draft
(
    user_id                bigint primary key references "user" (id),
    pending_competition_id bigint      not null references competition (id) on delete cascade,
    created_at             timestamptz not null default now()
);

-- +goose Down
drop table competition_merge_draft;
