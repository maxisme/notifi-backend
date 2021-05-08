alter table notifications
    alter column time type timestamp using time::timestamp;

alter table notifications
    alter column time set default CURRENT_TIMESTAMP;

