-- +goose Up
alter table training add column rounds smallint;

-- +goose Down
alter table training drop column rounds;
