alter table notifications
    add UUID varchar(255) null;

create unique index notifications_uuid_uindex
    on notifications (UUID);