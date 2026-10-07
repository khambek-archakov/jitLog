-- +goose Up
alter table training_draft drop column notes;

-- +goose Down
alter table training_draft add column notes text;
