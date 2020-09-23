create table if not exists users
(
    id               int auto_increment
        primary key,
    created          timestamp  default CURRENT_TIMESTAMP not null,
    credentials      varchar(60)                          null,
    credential_key   varchar(200)                         null,
    last_login       varchar(100)                         null,
    is_connected     tinyint(1) default 0                 not null,
    app_version      varchar(20)                          null,
    notification_cnt int(100)   default 0                 not null,
    UUID             varchar(60)                          null,
    public_key       varchar(1000)                        null,
    constraint UUID
        unique (UUID),
    constraint credentials
        unique (credentials)
);

create table if not exists notifications
(
    id          int auto_increment
        primary key,
    time        timestamp default CURRENT_TIMESTAMP not null,
    credentials varchar(60)                         null,
    title       varchar(1000)                       not null,
    message     varchar(10000)                      not null,
    image       varchar(4000)                       not null,
    link        varchar(4000)                       not null,
    UUID        varchar(255)                        not null,
    constraint UUID
        unique (UUID),
    constraint credentials
        foreign key (credentials) references users (credentials)
);

