---- Types ----

-- User Status
CREATE TYPE user_status AS ENUM ('active','inactive','banned','applying', 'deleted');

---- Tables ----

-- Users Table
CREATE TABLE IF NOT EXISTS goat.public.users
(
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        TEXT        NOT NULL,
    email       TEXT        NOT NULL UNIQUE,
    password    TEXT        NOT NULL,
    user_status user_status NOT NULL,
    user_ip     TEXT        NOT NULL,
    avatar_name TEXT        NULL     DEFAULT NULL,
    created_at  TIMESTAMP   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMP   NOT NULL DEFAULT now()
);

-- Roles Table
CREATE TABLE IF NOT EXISTS goat.public.roles
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    type       TEXT      NOT NULL UNIQUE, -- 'admin', 'vendor', 'user', 'guest'
    creator    BIGINT REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Roles Users Table
CREATE TABLE IF NOT EXISTS goat.public.users_roles
(
    user_id    BIGINT REFERENCES users (id) ON DELETE CASCADE,
    role_id    BIGINT REFERENCES roles (id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);
