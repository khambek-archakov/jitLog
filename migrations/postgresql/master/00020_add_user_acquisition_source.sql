-- +goose Up
alter table "user" add column acquisition_source text;

-- +goose Down
alter table "user" drop column acquisition_source;
