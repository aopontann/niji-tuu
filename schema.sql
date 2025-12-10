-- DataGripで自動生成したDDL
create table categories
(
    id         varchar(30)                        not null
        primary key,
    name       varchar(100)                       not null,
    created_at datetime default CURRENT_TIMESTAMP not null,
    updated_at datetime default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP
);

create table keywords
(
    role_id        varchar(30)                        not null
        primary key,
    name           varchar(100)                       not null,
    category_id    varchar(30)                        not null,
    channel_id     varchar(30)                        not null,
    inclusion_list varchar(1000)     default ''                not null,
    exclusion_list varchar(1000)     default ''                not null,
    created_at     datetime default CURRENT_TIMESTAMP not null,
    updated_at     datetime default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP,
    constraint name
        unique (name)
);

create table users
(
    token      varchar(300)                         not null
        primary key,
    song       tinyint(1) default 0                 not null,
    info       tinyint(1) default 0                 not null,
    created_at datetime   default CURRENT_TIMESTAMP not null,
    updated_at datetime   default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP
);

create table videos
(
    id                   varchar(11)                        not null
        primary key,
    title                varchar(100)                       not null,
    duration             varchar(20)                        not null,
    content              varchar(20)                        not null,
    scheduled_start_time datetime                           not null,
    created_at           datetime default CURRENT_TIMESTAMP not null,
    updated_at           datetime default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP
);

create table vtubers
(
    id                  varchar(24)                            not null
        primary key,
    name                varchar(100)                           not null,
    item_count          int          default 0                 not null,
    playlist_latest_url varchar(300) default ''                not null,
    created_at          datetime     default CURRENT_TIMESTAMP not null,
    updated_at          datetime     default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP
);

