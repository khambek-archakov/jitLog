-- +goose Up
create table training_edit_draft
(
    user_id     bigint primary key references "user" (id),
    training_id bigint      not null references training (id) on delete cascade,
    field       smallint    not null,
    created_at  timestamptz not null default now()
);

-- +goose Down
drop table training_edit_draft;
