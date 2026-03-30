-- H-1: Migrate TIMESTAMP → TIMESTAMPTZ to ensure correct timezone handling
-- across all deployment environments. Without this, CountByRoomAfter time
-- comparisons can silently mis-count unread messages in non-UTC deployments.

ALTER TABLE public.chat_rooms
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

ALTER TABLE public.chat_members
    ALTER COLUMN last_read_at TYPE TIMESTAMPTZ USING last_read_at AT TIME ZONE 'UTC',
    ALTER COLUMN joined_at    TYPE TIMESTAMPTZ USING joined_at    AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at   TYPE TIMESTAMPTZ USING updated_at   AT TIME ZONE 'UTC',
    ALTER COLUMN deleted_at   TYPE TIMESTAMPTZ USING deleted_at   AT TIME ZONE 'UTC';

ALTER TABLE public.chat_records
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- H-5: Add index to support cursor-based pagination by id on chat_records.
-- FindByRoomBefore and FindLatestByRoom both use ORDER BY id DESC,
-- but the existing index is (room_id, created_at DESC) which does not
-- cover the id range scan.
CREATE INDEX IF NOT EXISTS idx_chat_records_room_id ON public.chat_records (room_id, id DESC);
