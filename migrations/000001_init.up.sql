create table if not exists users
(
    id               SERIAL
        primary key,
    created          timestamp default CURRENT_TIMESTAMP not null,
    credentials      varchar(60)                         null,
    credential_key   varchar(200)                        null,
    last_login       varchar(100)                        null,
    is_connected     boolean   default false             not null,
    app_version      varchar(20)                         null,
    notification_cnt int       default 0                 not null,
    UUID             varchar(60)                         null,
    public_key       varchar(1000)                       null,
    constraint UUID
        unique (UUID),
    constraint credentials
        unique (credentials)
);

create table if not exists notifications
(
    UUID        uuid primary key,
    time        timestamp default CURRENT_TIMESTAMP not null,
    credentials varchar(60)                         null,
    title       text                                not null,
    message     text                                not null,
    image       text                                not null,
    link        text                                not null,
    constraint credentials
        foreign key (credentials) references users (credentials)
);

