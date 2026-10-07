-- +goose Up
alter table "user" add column city text;

-- +goose Down
alter table "user" drop column city;
