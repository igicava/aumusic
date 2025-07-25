create table if not exists users (
    id uuid primary key not null unique default gen_random_uuid(),
    name text not null not null unique,
    email text not null unique,
    password text not null
);

create table if not exists tracks (
    id uuid primary key unique not null,
    user_id uuid not null references users(id),
    name text not null not null,
    artist text not null not null,
    album text not null not null,
    size int not null not null,
    path text not null not null,
    mod_time timestamp not null
);

create table if not exists playlists (
    id uuid primary key unique not null,
    user_id uuid not null references users(id),
    name text not null not null unique
);

create table if not exists playlist_tracks (
    id uuid primary key unique not null,
    playlist_id uuid not null references playlists(id),
    track_id uuid not null references tracks(id)
);
