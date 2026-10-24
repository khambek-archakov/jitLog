-- +goose Up
alter table reminder_sent rename to reminder_log;
alter table reminder_log rename column week_start to ref_date;
alter table reminder_log add column kind smallint not null default 1; -- 1 weekly, 2 post_session
alter table reminder_log add column slot_id bigint not null default 0; -- schedule_slot.id, 0 for weekly
alter table reminder_log add column message_id bigint;
alter table reminder_log drop constraint reminder_sent_pkey;
alter table reminder_log add primary key (user_id, kind, ref_date, slot_id);

alter table schedule_slot add column duration_min smallint not null default 90
    check (duration_min between 15 and 300);

-- schedule_edit_draft previously only ever held a pending time edit (the
-- only field that could need free text) — now duration's own "Другое"
-- can too, so it needs the same field disambiguator training_edit_draft
-- already has. Default 1 (time) is exactly correct for any in-flight
-- draft row at migration time, since duration-editing didn't exist yet.
alter table schedule_edit_draft add column field smallint not null default 1; -- 1 time, 2 duration

-- +goose Down
alter table schedule_edit_draft drop column field;
alter table schedule_slot drop column duration_min;

alter table reminder_log drop constraint reminder_log_pkey;
alter table reminder_log drop column message_id;
alter table reminder_log drop column slot_id;
alter table reminder_log drop column kind;
alter table reminder_log rename column ref_date to week_start;
alter table reminder_log rename to reminder_sent;
alter table reminder_sent add constraint reminder_sent_pkey primary key (user_id, week_start);
