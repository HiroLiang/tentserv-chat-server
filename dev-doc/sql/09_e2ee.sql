-- ============================================================
-- [EN] File: 09_e2ee.sql  |  Execution order: 9 of 10
--      Creates the end-to-end encryption (Signal Protocol X3DH) layer:
--      user_identity_keys, user_signed_pre_keys, user_one_time_pre_keys,
--      member_sender_keys, sender_key_distributions, sender_key_requests
--      tables with their indexes.
--
--      NOTE: sender_key_requests was previously a separate migration
--      file (migrate_add_sender_key_requests.sql). It is now integrated
--      here. That migration file is no longer needed.
--
-- [中] 檔案：09_e2ee.sql  |  執行順序：第 9 支（共 10 支）
--      建立端對端加密（Signal Protocol X3DH）層：
--      user_identity_keys、user_signed_pre_keys、user_one_time_pre_keys、
--      member_sender_keys、sender_key_distributions、sender_key_requests
--      資料表及其索引。
--
--      注意：sender_key_requests 原為獨立遷移檔案
--      （migrate_add_sender_key_requests.sql），現已整合於此。
--
-- [日] ファイル：09_e2ee.sql  |  実行順：9/10
--      エンドツーエンド暗号化（Signal Protocol X3DH）層を作成する：
--      user_identity_keys、user_signed_pre_keys、user_one_time_pre_keys、
--      member_sender_keys、sender_key_distributions、sender_key_requests
--      テーブルとそのインデックス。
--
--      注意：sender_key_requests は以前は独立したマイグレーションファイル
--      （migrate_add_sender_key_requests.sql）だったが、ここに統合された。
--
-- Dependencies | 依賴 | 依存: 01_account.sql (users), 02_device.sql (devices),
--                              07_chat.sql (chat_members)
-- Creates      | 建立 | 作成:
--   TABLE user_identity_keys
--   TABLE user_signed_pre_keys     + INDEX idx_user_signed_pre_keys_active
--   TABLE user_one_time_pre_keys   + INDEX idx_user_one_time_pre_keys_lookup
--   TABLE member_sender_keys       + INDEX idx_member_sender_keys_member
--   TABLE sender_key_distributions + INDEX idx_sender_key_distributions_sender
--                                  + INDEX idx_sender_key_distributions_receiver
--   TABLE sender_key_requests      + INDEX idx_sender_key_requests_provider
-- ============================================================


-- ============================================================
-- SECTION 1: IDENTITY KEYS
-- [EN] One identity key bundle per (user, device) pair.
--      public_key   = 32-byte Curve25519 DH key (for X3DH)
--      sign_public_key = 32-byte Ed25519 signing key (for SPK verification)
--      fingerprint  = hex(SHA-256(public_key)) displayed in safety number UIs
-- [中] 每個（user, device）對有一組身份金鑰。
--      public_key      = 32 位元組 Curve25519 DH 金鑰（用於 X3DH）
--      sign_public_key = 32 位元組 Ed25519 簽名金鑰（用於 SPK 驗證）
--      fingerprint     = hex(SHA-256(public_key))，顯示於安全號碼 UI
-- [日] (user, device) ペアごとに 1 つの identity key バンドル。
--      public_key      = 32 バイト Curve25519 DH 鍵（X3DH 用）
--      sign_public_key = 32 バイト Ed25519 署名鍵（SPK 検証用）
--      fingerprint     = hex(SHA-256(public_key))、安全番号 UI に表示
-- ============================================================

-- User identity keys | 使用者身份金鑰 | ユーザー身分鍵
CREATE TABLE IF NOT EXISTS public.user_identity_keys
(
    id              BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id         BIGINT    NOT NULL REFERENCES public.users (id) ON DELETE CASCADE,
    device_id       UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    public_key      BYTEA     NOT NULL, -- 32-byte Curve25519 public key
    sign_public_key BYTEA     NOT NULL, -- 32-byte Ed25519 signing public key
    fingerprint     TEXT      NOT NULL, -- hex(SHA-256(public_key)) for display
    uploaded_at     TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (user_id, device_id)
);


-- ============================================================
-- SECTION 2: SIGNED PRE-KEYS
-- [EN] Medium-term pre-keys rotated periodically.
--      Signed by the identity key to prove authenticity.
--      is_active flags the current key; others are kept for
--      in-flight session resolution.
-- [中] 定期輪換的中期預金鑰。由身份金鑰簽名以證明真實性。
--      is_active 標記當前金鑰；其餘金鑰保留以供進行中的會話解析。
-- [日] 定期ローテーションされる中期プリキー。
--      identity key で署名して真正性を証明する。
--      is_active が現在の鍵をフラグし、他は進行中のセッション解決のために保持する。
-- ============================================================

-- Signed pre-keys | 已簽名預金鑰 | 署名済みプリキー
CREATE TABLE IF NOT EXISTS public.user_signed_pre_keys
(
    id         BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id    BIGINT    NOT NULL REFERENCES public.users (id) ON DELETE CASCADE,
    device_id  UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    key_id     INT       NOT NULL, -- client-assigned uint32 identifier
    public_key BYTEA     NOT NULL, -- 32-byte Curve25519 public key
    signature  BYTEA     NOT NULL, -- 64-byte Ed25519 signature over public_key
    is_active  BOOLEAN   NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    expires_at TIMESTAMP,
    UNIQUE (user_id, device_id, key_id)
);

-- Index for fast active-key lookup | 快速查詢有效金鑰的索引 | アクティブキー高速検索インデックス
CREATE INDEX IF NOT EXISTS idx_user_signed_pre_keys_active
    ON public.user_signed_pre_keys (user_id, device_id, is_active);


-- ============================================================
-- SECTION 3: ONE-TIME PRE-KEYS
-- [EN] Ephemeral X3DH pre-keys consumed atomically on session
--      initiation. Each key_id is consumed once and deleted.
-- [中] 在會話建立時原子消費的一次性 X3DH 預金鑰。每個 key_id 使用一次後刪除。
-- [日] セッション開始時に原子的に消費されるエフェメラル X3DH プリキー。
--      各 key_id は一度消費されたら削除される。
-- ============================================================

-- One-time pre-keys | 一次性預金鑰 | ワンタイムプリキー
CREATE TABLE IF NOT EXISTS public.user_one_time_pre_keys
(
    id          BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id     BIGINT    NOT NULL REFERENCES public.users (id) ON DELETE CASCADE,
    device_id   UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    key_id      INT       NOT NULL, -- client-assigned uint32 identifier
    public_key  BYTEA     NOT NULL, -- 32-byte Curve25519 public key
    uploaded_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (user_id, device_id, key_id)
);

-- Index for key-bundle fetch | 金鑰包取得的索引 | 鍵バンドル取得インデックス
CREATE INDEX IF NOT EXISTS idx_user_one_time_pre_keys_lookup
    ON public.user_one_time_pre_keys (user_id, device_id);


-- ============================================================
-- SECTION 4: MEMBER SENDER KEYS
-- [EN] Per-member AES-256-GCM sender key for group/direct E2EE.
--      distribution_message is an opaque sealed blob:
--        base64(JSON.stringify(X3DH InitialMessage encrypted for recipient))
--      sender_key_version is generated by Rust and currently uses a millisecond-scale version.
--      chain_id mirrors that version for legacy compatibility. Server stores and relays
--      without ever parsing the blob.
-- [中] 每個成員用於群組/直接 E2EE 的 AES-256-GCM sender key。
--      distribution_message 是不透明的封存 blob：
--        base64(JSON.stringify(為接收者加密的 X3DH InitialMessage))
--      sender_key_version 由 Rust 產生，目前使用毫秒級版本號；
--      chain_id 僅為相容舊流程的鏡像欄位。伺服器儲存並轉送，從不解析 blob。
-- [日] グループ/ダイレクト E2EE 用のメンバーごとの AES-256-GCM sender key。
--      distribution_message は不透明な封印 blob：
--        base64(JSON.stringify(受信者向けに暗号化された X3DH InitialMessage))
--      sender_key_version は Rust が生成するミリ秒単位のバージョン値であり、
--      chain_id は後方互換のためのミラー列である。
--      サーバーは blob を解析せずに保存・中継する。
-- ============================================================

-- Member sender key metadata | 成員發送方金鑰中繼資料 | メンバー送信者鍵メタデータ
CREATE TABLE IF NOT EXISTS public.member_sender_keys
(
    id                   BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    chat_member_id       BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    sender_device_id     UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE, -- metadata only; canonical ownership is chat_member_id + sender_key_version
    chain_id             BIGINT    NOT NULL DEFAULT 1, -- mirrors sender_key_version for legacy compatibility
    sender_key_version   BIGINT    NOT NULL,           -- latest sender key version created by Rust
    key_fingerprint      TEXT,                         -- optional future debug / auditing metadata
    created_at           TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (chat_member_id, sender_key_version)
);

-- Index for member-key lookup | 成員金鑰查詢索引 | メンバーキー検索インデックス
CREATE INDEX IF NOT EXISTS idx_member_sender_keys_member
    ON public.member_sender_keys (chat_member_id);


-- ============================================================
-- SECTION 5: SENDER KEY RECEIPTS
-- [EN] Canonical receiver-side possession state for sender keys.
--      This is the authoritative answer for whether a receiver device
--      already holds a sender member/device's latest sender_key_version.
-- [中] sender key 接收端持有狀態的唯一權威來源。
--      用來判斷某個 receiver device 是否已持有某個 sender member/device
--      的最新 sender_key_version。
-- [日] sender key を受信側がすでに保有しているかを示す単一の正準状態。
-- ============================================================

CREATE TABLE IF NOT EXISTS public.sender_key_receipts
(
    id                 BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    sender_member_id   BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    sender_device_id   UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE, -- last provider device metadata
    receiver_member_id BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE, -- room/member metadata
    receiver_device_id UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    sender_key_version BIGINT    NOT NULL,
    source             TEXT      NOT NULL,
    updated_at         TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (sender_member_id, receiver_device_id)
);

CREATE INDEX IF NOT EXISTS idx_sender_key_receipts_sender
    ON public.sender_key_receipts (sender_member_id);

CREATE INDEX IF NOT EXISTS idx_sender_key_receipts_receiver
    ON public.sender_key_receipts (receiver_device_id);


-- ============================================================
-- SECTION 6: SENDER KEY DISTRIBUTIONS
-- [EN] Transport queue for sender-key copies.
--      This table tracks delivery state only; possession is derived
--      from sender_key_receipts after the receiver consumes a copy.
-- [中] sender key 的傳輸佇列，只代表投遞狀態；
--      真正是否已持有由 sender_key_receipts 判斷。
-- [日] sender key の配送キュー。受信側が実際に保有済みかどうかは
--      sender_key_receipts で判断する。
-- ============================================================

CREATE TABLE IF NOT EXISTS public.sender_key_distributions
(
    id                   BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    sender_member_id     BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    sender_device_id     UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    receiver_member_id   BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    receiver_device_id   UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    sender_key_version   BIGINT    NOT NULL,
    distribution_message BYTEA     NOT NULL,
    status               TEXT      NOT NULL DEFAULT 'available',
    chain_id             BIGINT    NOT NULL DEFAULT 1, -- legacy compatibility; mirrors sender_key_version
    distributed_at       TIMESTAMP NOT NULL DEFAULT now(),
    consumed_at          TIMESTAMP,
    failed_at            TIMESTAMP,
    UNIQUE (sender_member_id, sender_device_id, receiver_member_id, receiver_device_id)
);

-- Index for "who has received my key" | 「誰已收到我的金鑰」索引 | 「誰が自分の鍵を受け取ったか」インデックス
CREATE INDEX IF NOT EXISTS idx_sender_key_distributions_sender
    ON public.sender_key_distributions (sender_member_id, sender_device_id);
-- Index for "whose key have I received" | 「我已收到誰的金鑰」索引 | 「誰の鍵を受け取ったか」インデックス
CREATE INDEX IF NOT EXISTS idx_sender_key_distributions_receiver
    ON public.sender_key_distributions (receiver_member_id, receiver_device_id);


-- ============================================================
-- SECTION 7: SENDER KEY REQUESTS
-- [EN] Records that requester_member_id needs provider_member_id
--      to upload their sender key.
--      Created when a member enters a room and pending_from_members ≠ ∅.
--      Fulfilled (and deleted) when the provider uploads their sender key.
--      Acts as a durable offline notification: on reconnect the server
--      calls NotifyPendingSenderKeyRequestsUseCase to replay pending rows.
-- [中] 記錄 requester_member_id 需要 provider_member_id 上傳其 sender key。
--      當成員進入房間且 pending_from_members ≠ ∅ 時建立。
--      provider 上傳 sender key 後標記完成（fulfilled）。
--      作為持久的離線通知：重新連線時伺服器呼叫
--      NotifyPendingSenderKeyRequestsUseCase 重新播放待處理列。
-- [日] requester_member_id が provider_member_id の sender key のアップロードを
--      必要としていることを記録する。
--      メンバーがルームに入り pending_from_members ≠ ∅ のときに作成される。
--      provider が sender key をアップロードすると fulfilled にマークされる（削除なし）。
--      永続的なオフライン通知として機能する：再接続時にサーバーが
--      NotifyPendingSenderKeyRequestsUseCase を呼び出して保留行を再生する。
-- ============================================================

-- Sender key requests | 發送方金鑰請求 | 送信者鍵リクエスト
CREATE TABLE IF NOT EXISTS public.sender_key_requests
(
    id                  BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    requester_member_id BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    requester_device_id UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    provider_member_id  BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    provider_device_id  UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    created_at          TIMESTAMP NOT NULL DEFAULT now(),
    fulfilled_at        TIMESTAMP,
    UNIQUE (requester_member_id, requester_device_id, provider_member_id, provider_device_id)
);

-- Partial index: only unfulfilled requests (fulfilled_at IS NULL = provider still needs to act)
-- 部分索引：僅未完成的請求（fulfilled_at IS NULL = provider 尚需動作）
-- 部分インデックス：未完了リクエストのみ（fulfilled_at IS NULL = プロバイダーがまだ対応が必要）
CREATE INDEX IF NOT EXISTS idx_sender_key_requests_provider
    ON public.sender_key_requests (provider_member_id, provider_device_id)
    WHERE fulfilled_at IS NULL;

-- ============================================================
-- SECTION 8: SELF SENDER KEY SYNC
-- [EN] Tracks the one active self sender-key bootstrap flow for a participant.
-- [中] 追蹤單一 participant 當前唯一一輪自有 sender key 補同步流程。
-- [日] participant ごとに同時 1 件だけ許可される自己 sender key 同期を追跡する。
-- ============================================================

CREATE TABLE IF NOT EXISTS public.participant_self_sender_key_syncs
(
    id                   BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    participant_id       BIGINT    NOT NULL REFERENCES public.participants (id) ON DELETE CASCADE,
    requester_device_id  UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    provider_device_id   UUID      REFERENCES public.devices (id) ON DELETE SET NULL,
    status               TEXT      NOT NULL,
    requested_at         TIMESTAMP NOT NULL DEFAULT now(),
    provider_claimed_at  TIMESTAMP,
    uploaded_at          TIMESTAMP,
    completed_at         TIMESTAMP,
    failed_at            TIMESTAMP,
    last_error           TEXT,
    updated_at           TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_participant_self_sender_key_syncs_status
    ON public.participant_self_sender_key_syncs (status);

CREATE INDEX IF NOT EXISTS idx_participant_self_sender_key_syncs_participant
    ON public.participant_self_sender_key_syncs (participant_id, id DESC);

CREATE TABLE IF NOT EXISTS public.self_sender_key_sync_distributions
(
    id                   BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    self_sender_key_sync_id BIGINT NOT NULL REFERENCES public.participant_self_sender_key_syncs (id) ON DELETE CASCADE,
    participant_id       BIGINT    NOT NULL REFERENCES public.participants (id) ON DELETE CASCADE,
    requester_device_id  UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    provider_device_id   UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    sender_member_id     BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    sender_device_id     UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    sender_key_version   BIGINT    NOT NULL,
    distribution_message BYTEA     NOT NULL,
    status               TEXT      NOT NULL DEFAULT 'available',
    created_at           TIMESTAMP NOT NULL DEFAULT now(),
    consumed_at          TIMESTAMP,
    failed_at            TIMESTAMP,
    UNIQUE (self_sender_key_sync_id, participant_id, requester_device_id, provider_device_id, sender_member_id, sender_device_id, sender_key_version)
);

CREATE INDEX IF NOT EXISTS idx_self_sender_key_sync_distributions_requester
    ON public.self_sender_key_sync_distributions (self_sender_key_sync_id, participant_id, requester_device_id, provider_device_id, status);
