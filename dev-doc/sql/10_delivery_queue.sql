-- ============================================================
-- [EN] File: 10_delivery_queue.sql  |  Execution order: 10 of 10
--      Creates the WebSocket delivery queue: delivery_queue table
--      and its indexes. Used to reliably deliver messages to
--      temporarily offline users by retrying on reconnect.
--
-- [中] 檔案：10_delivery_queue.sql  |  執行順序：第 10 支（共 10 支）
--      建立 WebSocket 投遞佇列：delivery_queue 資料表及其索引。
--      用於在使用者暫時離線時可靠地投遞訊息，重新連線後自動重試。
--
-- [日] ファイル：10_delivery_queue.sql  |  実行順：10/10
--      WebSocket 配信キューを作成する：delivery_queue テーブルとそのインデックス。
--      一時的にオフラインのユーザーへのメッセージを再接続時に
--      確実に再送するために使用する。
--
-- Dependencies | 依賴 | 依存: 01_account.sql (users)
-- Creates      | 建立 | 作成:
--   TABLE delivery_queue
--   INDEX idx_delivery_queue_user_status, idx_delivery_queue_status_create_at
-- ============================================================


-- ============================================================
-- SECTION 1: TABLES
-- [EN] delivery_queue stores serialized WebSocket payloads for
--      users who were offline when the event was generated.
--      payload_type is a string tag (e.g. 'chat_message', 'e2ee.sender_key_needed').
--      status: 'pending' → 'delivered'; the retry scheduler polls
--      pending rows and replays them on reconnect.
-- [中] delivery_queue 儲存在事件產生時使用者離線的序列化 WebSocket 負載。
--      payload_type 為字串標籤（例如 'chat_message'、'e2ee.sender_key_needed'）。
--      status：'pending' → 'delivered'；重試排程器輪詢待處理列
--      並在重新連線時重播。
-- [日] delivery_queue は、イベント生成時にオフラインだったユーザーへの
--      シリアライズされた WebSocket ペイロードを格納する。
--      payload_type は文字列タグ（例：'chat_message'、'e2ee.sender_key_needed'）。
--      status：'pending' → 'delivered'；リトライスケジューラーが
--      保留行をポーリングし、再接続時に再生する。
-- ============================================================

-- WebSocket delivery queue | WebSocket 投遞佇列 | WebSocket 配信キュー
CREATE TABLE public.delivery_queue
(
    id           BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id      BIGINT    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    payload_type TEXT      NOT NULL,
    payload      BYTEA     NOT NULL,
    status       TEXT      NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMP NOT NULL DEFAULT now(),
    delivered_at TIMESTAMP
);


-- ============================================================
-- SECTION 2: INDEXES
-- [EN] idx_delivery_queue_user_status: find pending messages for
--      a specific user on reconnect (most critical query path).
--      idx_delivery_queue_status_create_at: global sweep for the
--      retry scheduler to find all stale pending messages ordered
--      by creation time.
-- [中] idx_delivery_queue_user_status：重新連線時查詢特定使用者的
--      待處理訊息（最關鍵的查詢路徑）。
--      idx_delivery_queue_status_create_at：重試排程器的全域掃描，
--      按建立時間排序尋找所有過期的待處理訊息。
-- [日] idx_delivery_queue_user_status：再接続時に特定ユーザーの
--      保留メッセージを検索する（最重要クエリパス）。
--      idx_delivery_queue_status_create_at：リトライスケジューラーが
--      作成時間順にすべての古い保留メッセージを検索するグローバルスキャン。
-- ============================================================

-- Per-user pending lookup | 每位使用者待處理查詢 | ユーザーごとの保留検索
CREATE INDEX idx_delivery_queue_user_status      ON delivery_queue (user_id, status);
-- Global retry sweep | 全域重試掃描 | グローバルリトライスキャン
CREATE INDEX idx_delivery_queue_status_create_at ON delivery_queue (status, created_at);
