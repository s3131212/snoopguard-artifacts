create table "User"
(
    access_hash            bigint,
    bot                    boolean,
    bot_chat_history       boolean,
    bot_info_version       integer,
    bot_inline_geo         boolean,
    bot_inline_placeholder text,
    bot_nochats            boolean,
    contact                boolean,
    deleted                boolean,
    first_name             text,
    id                     bigint not null
        primary key,
    is_self                boolean,
    lang_code              text,
    last_name              text,
    min                    boolean,
    mutual_contact         boolean,
    phone                  text,
    restricted             boolean,
    retrieved_utc          timestamp,
    scam                   boolean,
    support                boolean,
    username               text
        unique,
    verified               boolean
);

alter table "User"
    owner to postgres;

create table "Channel"
(
    access_hash        bigint,
    broadcast          boolean,
    creator            boolean,
    date               timestamp with time zone,
    has_geo            boolean,
    has_link           boolean,
    id                 bigint not null
        constraint channel_pkey
            primary key,
    megagroup          boolean,
    min                boolean,
    participants_count integer,
    restricted         boolean,
    scam               boolean,
    signatures         boolean,
    title              text,
    username           text,
    verified           boolean,
    version            integer
);

alter table "Channel"
    owner to postgres;

create table "ChannelFull"
(
    about                 text,
    admins_count          integer,
    available_min_id      integer,
    banned_count          integer,
    can_set_location      boolean,
    can_set_stickers      boolean,
    can_set_username      boolean,
    can_view_participants boolean,
    can_view_stats        boolean,
    folder_id             integer,
    hidden_prehistory     boolean,
    id                    bigint not null
        constraint channelfull_pkey
            primary key,
    kicked_count          integer,
    linked_chat_id        bigint,
    migrated_from_chat_id bigint,
    migrated_from_max_id  integer,
    online_count          integer,
    participants_count    integer,
    pinned_msg_id         integer,
    pts                   integer,
    read_inbox_max_id     integer,
    read_outbox_max_id    integer,
    unread_count          integer
);

alter table "ChannelFull"
    owner to postgres;

create table "Message"
(
    date            timestamp with time zone,
    edit_date       timestamp with time zone,
    from_id         bigint
        constraint fk_from_id
            references "User",
    from_scheduled  boolean,
    grouped_id      bigint,
    id              integer not null,
    legacy          boolean,
    media_unread    boolean,
    mentioned       boolean,
    message         text,
    out             boolean,
    post            boolean,
    post_author     text,
    reply_to_msg_id integer,
    retrieved_utc   timestamp,
    silent          boolean,
    to_channel_id   bigint  not null
        constraint fk_to_channel_id
            references "Channel",
    via_bot_id      bigint,
    views           integer,
    constraint message_pkey
        primary key (id, to_channel_id)
);

alter table "Message"
    owner to postgres;

