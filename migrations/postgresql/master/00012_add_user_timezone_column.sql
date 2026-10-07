-- +goose Up
alter table "user" add column timezone text;

-- +goose Down
alter table "user" drop column timezone;
