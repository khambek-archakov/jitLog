-- +goose Up
create table schedule_edit_draft
(
    user_id    bigint primary key references "user" (id),
    slot_id    bigint      not null references schedule_slot (id) on delete cascade,
    created_at timestamptz not null default now()
);

-- +goose Down
drop table schedule_edit_draft;
