---- Types ----

-- User Status
CREATE TYPE account_status AS ENUM ('active','inactive','banned','applying', 'deleted');

---- Tables ----

-- Accounts
CREATE TABLE IF NOT EXISTS goat.public.accounts
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id  UUID           NOT NULL UNIQUE,
    email      VARCHAR(255)   NOT NULL UNIQUE,
    account    VARCHAR(225)   NOT NULL UNIQUE,
    password   TEXT           NOT NULL,
    status     account_status NOT NULL DEFAULT 'applying',
    user_limit INTEGER        NOT NULL DEFAULT 1,
    created_at TIMESTAMP      NOT NULL DEFAULT now(),
    updated_at TIMESTAMP      NOT NULL DEFAULT now()
);

-- Users
CREATE TABLE IF NOT EXISTS goat.public.users
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id BIGINT REFERENCES accounts (id) ON DELETE CASCADE,
    name       TEXT      NOT NULL,
    avatar     TEXT      NULL     DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Roles Table
CREATE TABLE IF NOT EXISTS goat.public.roles
(
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code        TEXT      NOT NULL UNIQUE, -- machine name
    name        TEXT      NOT NULL,
    description TEXT,
    created_by  BIGINT    REFERENCES users (id) ON DELETE SET NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT now(),
    updated_at  TIMESTAMP NOT NULL DEFAULT now()
);

-- Roles Users Table
CREATE TABLE IF NOT EXISTS goat.public.users_roles
(
    user_id BIGINT REFERENCES users (id) ON DELETE CASCADE,
    role_id BIGINT REFERENCES roles (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

---- Trigger ----

-- Limit user number of single account
CREATE OR REPLACE FUNCTION limit_user_per_account()
    RETURNS trigger AS
$$
DECLARE
    current_count INTEGER;
    max_limit     INTEGER;
BEGIN
    SELECT COUNT(*)
    INTO current_count
    FROM users
    WHERE account_id = NEW.account_id;

    SELECT user_limit
    INTO max_limit
    FROM accounts
    WHERE id = NEW.account_id;

    IF current_count >= max_limit THEN
        RAISE EXCEPTION 'User limit exceeded for this account';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_limit_user
    BEFORE INSERT
    ON goat.public.users
    FOR EACH ROW
EXECUTE FUNCTION limit_user_per_account();

---- Default data ----
INSERT INTO goat.public.accounts (public_id, email, account, password, status)
VALUES (gen_random_uuid(),
        'hiro@gmail.com',
        'hiro',
        'eRB2soJ8fMdiEoUmOafq1Q:GhpLtQK6dDnFDumbl1MNs6NU9yOt5QpuXGpIKBG7Az0',
        'active');


INSERT INTO goat.public.users (account_id, name)
VALUES (1, 'Hiro Liang');

INSERT INTO goat.public.roles (code, name, description, created_by)
VALUES ('admin', 'Administrator', 'Manager of this platform', 1),
       ('user', 'User', 'Authorized user', 1),
       ('vendor', 'Vendor', 'Vendor of service', 1),
       ('client', 'Client', 'Unauthed client', 1);

INSERT INTO goat.public.users_roles (user_id, role_id)
VALUES (1, 1),
       (1, 2),
       (1, 3);
