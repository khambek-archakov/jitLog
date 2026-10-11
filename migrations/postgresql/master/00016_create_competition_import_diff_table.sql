-- +goose Up
create table competition_import_diff
(
    competition_id     bigint primary key references competition (id) on delete cascade,
    proposed_date      date        not null,
    proposed_end_date  date,
    proposed_city      text,
    created_at         timestamptz not null default now(),
    check (proposed_end_date is null or proposed_end_date >= proposed_date)
);

-- +goose Down
drop table competition_import_diff;
