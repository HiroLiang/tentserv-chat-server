-- ============================================================
-- [EN] File: 03_email.sql  |  Execution order: 3 of 10
--      Creates the email logging layer: email_status enum and
--      email_logs table for tracking outbound transactional emails.
--
-- [中] 檔案：03_email.sql  |  執行順序：第 3 支（共 10 支）
--      建立電子郵件日誌層：email_status 列舉及 email_logs 資料表，
--      用於追蹤對外寄送的交易型郵件。
--
-- [日] ファイル：03_email.sql  |  実行順：3/10
--      メールログ層を作成する：email_status 列挙と
--      外部送信トランザクションメールを追跡する email_logs テーブル。
--
-- Dependencies | 依賴 | 依存: (none — standalone)
-- Creates      | 建立 | 作成:
--   TYPE  email_status
--   TABLE email_logs
-- ============================================================


-- ============================================================
-- SECTION 1: ENUM TYPES
-- [EN] Lifecycle states of an outbound email (sending → sent / failed).
-- [中] 對外郵件的生命週期狀態（發送中 → 已發送 / 失敗）。
-- [日] 送信メールのライフサイクルステータス（送信中 → 送信済み / 失敗）。
-- ============================================================

CREATE TYPE email_status AS ENUM ('sending', 'sent', 'failed');


-- ============================================================
-- SECTION 2: TABLES
-- [EN] email_logs persists every outbound email attempt,
--      including the external provider ID for reconciliation.
-- [中] email_logs 記錄每次對外郵件傳送嘗試，
--      包含外部供應商 ID 以便對帳。
-- [日] email_logs はすべての外部メール送信試行を永続化し、
--      外部プロバイダー ID で照合できるようにする。
-- ============================================================

-- Email send log | 郵件傳送日誌 | メール送信ログ
CREATE TABLE IF NOT EXISTS public.email_logs
(
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sender      VARCHAR(255) NOT NULL,
    recipients  TEXT         NOT NULL,
    subject     TEXT         NOT NULL,
    status      email_status NOT NULL,
    external_id TEXT         NULL,
    created_at  TIMESTAMP    NOT NULL DEFAULT now()
);
