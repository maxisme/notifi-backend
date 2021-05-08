alter table notifications
    alter column time type varchar(50) using time::varchar(50);

alter table notifications
    alter column time drop default;

