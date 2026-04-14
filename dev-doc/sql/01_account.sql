-- ============================================================
-- [EN] File: 01_account.sql  |  Execution order: 1 of 10
--      Creates the core account/user layer: account_status enum,
--      accounts, users, roles, users_roles tables, plus the
--      per-account user-count trigger, and seeds default data.
--
-- [中] 檔案：01_account.sql  |  執行順序：第 1 支（共 10 支）
--      建立帳號與使用者核心層：account_status 列舉、accounts、
--      users、roles、users_roles 資料表、帳號使用者數量限制觸發器，
--      並插入預設資料。
--
-- [日] ファイル：01_account.sql  |  実行順：1/10
--      アカウント・ユーザーコア層を作成する：account_status 列挙、
--      accounts、users、roles、users_roles テーブル、
--      アカウントごとのユーザー数制限トリガー、初期データの挿入。
--
-- Dependencies | 依賴 | 依存: (none — base layer)
-- Creates      | 建立 | 作成:
--   TYPE  account_status
--   TABLE accounts, users, roles, users_roles
--   FUNCTION limit_user_per_account()
--   TRIGGER trg_limit_user ON users
-- ============================================================


-- ============================================================
-- SECTION 1: ENUM TYPES
-- [EN] Define account lifecycle status values.
-- [中] 定義帳號生命週期狀態列舉。
-- [日] アカウントのライフサイクルステータスを定義する。
-- ============================================================

CREATE TYPE account_status AS ENUM ('active', 'inactive', 'banned', 'applying', 'deleted');


-- ============================================================
-- SECTION 2: TABLES
-- [EN] Core account and user tables. accounts is the root entity;
--      users, roles, and users_roles all reference it.
-- [中] 帳號與使用者核心資料表。accounts 為根實體；
--      users、roles、users_roles 均依賴它。
-- [日] アカウント・ユーザーのコアテーブル。accounts がルートエンティティ；
--      users、roles、users_roles はすべてこれを参照する。
-- ============================================================

-- Accounts | 帳號 | アカウント
CREATE TABLE IF NOT EXISTS public.accounts
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

-- Users | 使用者 | ユーザー
CREATE TABLE IF NOT EXISTS public.users
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id BIGINT REFERENCES accounts (id) ON DELETE CASCADE,
    name       TEXT      NOT NULL,
    avatar     TEXT      NULL     DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Roles | 角色 | ロール
CREATE TABLE IF NOT EXISTS public.roles
(
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code        TEXT      NOT NULL UNIQUE, -- machine name | 機器名稱 | マシン名
    name        TEXT      NOT NULL,
    description TEXT,
    created_by  BIGINT    REFERENCES users (id) ON DELETE SET NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT now(),
    updated_at  TIMESTAMP NOT NULL DEFAULT now()
);

-- User–Role join table | 使用者角色關聯表 | ユーザーロール中間テーブル
CREATE TABLE IF NOT EXISTS public.users_roles
(
    user_id BIGINT REFERENCES users (id) ON DELETE CASCADE,
    role_id BIGINT REFERENCES roles (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);


-- ============================================================
-- SECTION 3: FUNCTION + TRIGGER
-- [EN] Prevent inserting a user when the account's user_limit is
--      already reached. Fires BEFORE INSERT on public.users.
-- [中] 當帳號已達 user_limit 時，阻止新增使用者。
--      在 public.users 的 INSERT 前觸發。
-- [日] アカウントの user_limit に達している場合、ユーザー挿入を防ぐ。
--      public.users の INSERT 前に発火する。
-- ============================================================

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
    ON public.users
    FOR EACH ROW
EXECUTE FUNCTION limit_user_per_account();


-- ============================================================
-- SECTION 4: DEFAULT DATA
-- [EN] Seed one admin account, one user, standard roles,
--      and assign admin + user + vendor roles to the seed user.
-- [中] 插入一個管理員帳號、一位使用者、標準角色，並指派角色給種子使用者。
-- [日] 管理者アカウント・ユーザー・標準ロールを挿入し、
--      シードユーザーにロールを割り当てる。
-- ============================================================

INSERT INTO public.accounts (public_id, email, account, password, status)
VALUES (gen_random_uuid(),
        'hiromichi.liang@gmail.com',
        'hiro',
        'ZuvwbAh4eVnvmIY6BTA+mA:x1YiQMC649WNmZJ6P2ab04+mg/OtH3k9vyb0xi9Zt2k',
        'active');

INSERT INTO public.users (account_id, name)
VALUES (1, 'Hiro Liang');

INSERT INTO public.roles (code, name, description, created_by)
VALUES ('admin', 'Administrator', 'Manager of this platform', 1),
       ('user', 'User', 'Authorized user', 1),
       ('vendor', 'Vendor', 'Vendor of service', 1),
       ('client', 'Client', 'Unauthed client', 1);

INSERT INTO public.users_roles (user_id, role_id)
VALUES (1, 1),
       (1, 2),
       (1, 3);
