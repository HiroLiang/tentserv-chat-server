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
--      chain_id increments on re-key. Server stores and relays
--      without ever parsing the blob.
-- [中] 每個成員用於群組/直接 E2EE 的 AES-256-GCM sender key。
--      distribution_message 是不透明的封存 blob：
--        base64(JSON.stringify(為接收者加密的 X3DH InitialMessage))
--      chain_id 在重新產生金鑰時遞增。伺服器儲存並轉送，從不解析 blob。
-- [日] グループ/ダイレクト E2EE 用のメンバーごとの AES-256-GCM sender key。
--      distribution_message は不透明な封印 blob：
--        base64(JSON.stringify(受信者向けに暗号化された X3DH InitialMessage))
--      chain_id は再キー時にインクリメントされる。
--      サーバーは blob を解析せずに保存・中継する。
-- ============================================================

-- Member sender keys | 成員發送方金鑰 | メンバー送信者鍵
CREATE TABLE IF NOT EXISTS public.member_sender_keys
(
    id                   BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    chat_member_id       BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    chain_id             INT       NOT NULL DEFAULT 1, -- incremented on re-key
    sender_key_public    BYTEA     NOT NULL,           -- public part for verification
    distribution_message BYTEA     NOT NULL,           -- sealed SenderKeyDistributionMessage
    created_at           TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (chat_member_id, chain_id)
);

-- Index for member-key lookup | 成員金鑰查詢索引 | メンバーキー検索インデックス
CREATE INDEX IF NOT EXISTS idx_member_sender_keys_member
    ON public.member_sender_keys (chat_member_id);


-- ============================================================
-- SECTION 5: SENDER KEY DISTRIBUTION ACKNOWLEDGEMENTS
-- [EN] Records the latest chain_id of sender_member_id's key that
--      receiver_member_id has fetched. Written by GET /api/e2ee/sender-keys.
--      Answers:
--        "Who has already received my latest key?" → pending_receivers
--        "Whose key haven't I fetched yet?" → pending_from_members
-- [中] 記錄 sender_member_id 的最新 chain_id 已被 receiver_member_id 取得。
--      由 GET /api/e2ee/sender-keys 寫入。
--      回答：
--        「誰已收到我最新的金鑰？」→ pending_receivers
--        「我尚未取得誰的金鑰？」→ pending_from_members
-- [日] sender_member_id の最新 chain_id を receiver_member_id が取得したことを記録する。
--      GET /api/e2ee/sender-keys によって書き込まれる。
--      回答：
--        「誰がすでに自分の最新鍵を受け取ったか？」→ pending_receivers
--        「自分はまだ誰の鍵を取得していないか？」→ pending_from_members
-- ============================================================

-- Sender key distribution acks | 發送方金鑰分發確認 | 送信者鍵配布確認
CREATE TABLE IF NOT EXISTS public.sender_key_distributions
(
    id                 BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    sender_member_id   BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    receiver_member_id BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    chain_id           INT       NOT NULL, -- chain_id of member_sender_keys that was fetched
    distributed_at     TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (sender_member_id, receiver_member_id)
);

-- Index for "who has received my key" | 「誰已收到我的金鑰」索引 | 「誰が自分の鍵を受け取ったか」インデックス
CREATE INDEX IF NOT EXISTS idx_sender_key_distributions_sender
    ON public.sender_key_distributions (sender_member_id);
-- Index for "whose key have I received" | 「我已收到誰的金鑰」索引 | 「誰の鍵を受け取ったか」インデックス
CREATE INDEX IF NOT EXISTS idx_sender_key_distributions_receiver
    ON public.sender_key_distributions (receiver_member_id);


-- ============================================================
-- SECTION 6: SENDER KEY REQUESTS
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
    provider_member_id  BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    created_at          TIMESTAMP NOT NULL DEFAULT now(),
    fulfilled_at        TIMESTAMP,
    UNIQUE (requester_member_id, provider_member_id)
);

-- Partial index: only unfulfilled requests (fulfilled_at IS NULL = provider still needs to act)
-- 部分索引：僅未完成的請求（fulfilled_at IS NULL = provider 尚需動作）
-- 部分インデックス：未完了リクエストのみ（fulfilled_at IS NULL = プロバイダーがまだ対応が必要）
CREATE INDEX IF NOT EXISTS idx_sender_key_requests_provider
    ON public.sender_key_requests (provider_member_id)
    WHERE fulfilled_at IS NULL;
