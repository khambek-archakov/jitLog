-- +goose Up
create table belt_promotion
(
    id          bigserial primary key,
    user_id     bigint      not null references "user" (id),
    belt        smallint    not null,
    promoted_at date        not null,
    created_at  timestamptz not null default now(),
    unique (user_id, belt)
);

create index belt_promotion_user_id_idx on belt_promotion (user_id);

-- +goose Down
drop table belt_promotion;
