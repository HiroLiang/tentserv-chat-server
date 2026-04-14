-- ============================================================
-- [EN] File: 02_device.sql  |  Execution order: 2 of 10
--      Creates the device and session layer: device_platform enum,
--      devices, accounts_devices, account_sessions, and
--      account_login_events tables.
--
-- [中] 檔案：02_device.sql  |  執行順序：第 2 支（共 10 支）
--      建立設備與會話層：device_platform 列舉、devices、
--      accounts_devices、account_sessions、account_login_events 資料表。
--
-- [日] ファイル：02_device.sql  |  実行順：2/10
--      デバイス・セッション層を作成する：device_platform 列挙、
--      devices、accounts_devices、account_sessions、
--      account_login_events テーブル。
--
-- Dependencies | 依賴 | 依存: 01_account.sql (accounts, users)
-- Creates      | 建立 | 作成:
--   TYPE  device_platform
--   TABLE devices, accounts_devices, account_sessions, account_login_events
-- ============================================================


-- ============================================================
-- SECTION 1: ENUM TYPES
-- [EN] Supported device platforms for client identification.
-- [中] 支援的設備平台，用於識別連線裝置。
-- [日] クライアント識別に使用するデバイスプラットフォームの列挙。
-- ============================================================

CREATE TYPE device_platform AS ENUM ('android', 'ios', 'windows', 'macos', 'linux', 'browser', 'unknown');
CREATE TYPE account_device_status AS ENUM ('pending_verification', 'pending_sync', 'syncing', 'ready');


-- ============================================================
-- SECTION 2: TABLES
-- [EN] devices is the root; accounts_devices links accounts to
--      their registered devices; account_sessions tracks active
--      refresh tokens; account_login_events is an audit log.
-- [中] devices 為根表；accounts_devices 連結帳號與已登錄設備；
--      account_sessions 追蹤有效的 refresh token；
--      account_login_events 為稽核日誌。
-- [日] devices がルート；accounts_devices はアカウントと登録デバイスをリンク；
--      account_sessions はアクティブな refresh token を追跡；
--      account_login_events は監査ログ。
-- ============================================================

-- Devices | 設備 | デバイス
CREATE TABLE devices
(
    id         UUID PRIMARY KEY,
    platform   device_platform NOT NULL,
    name       VARCHAR(255)    NOT NULL,
    created_at TIMESTAMP       NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP       NOT NULL DEFAULT NOW()
);

-- Account's registered devices | 帳號已登錄設備 | アカウントの登録デバイス
CREATE TABLE accounts_devices
(
    account_id   BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    device_id    UUID   NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    status       account_device_status NOT NULL DEFAULT 'ready',
    last_ip      INET,
    last_seen_at TIMESTAMP,
    PRIMARY KEY (account_id, device_id)
);

-- Active sessions (refresh token store) | 有效會話（Refresh Token 儲存）| アクティブセッション（リフレッシュトークン保管）
CREATE TABLE account_sessions
(
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id         BIGINT    NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    user_id            BIGINT    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    device_id          UUID      NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    refresh_token_hash TEXT      NOT NULL UNIQUE,
    expires_at         TIMESTAMP NOT NULL,
    revoked            BOOLEAN   NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMP NOT NULL DEFAULT now(),
    last_used_at       TIMESTAMP
);

-- Login audit log | 登入事件稽核日誌 | ログイン監査ログ
CREATE TABLE account_login_events
(
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id  BIGINT    NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    device_uuid UUID      NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    ip_address  INET,
    user_agent  TEXT,
    success     BOOLEAN   NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT now()
);
