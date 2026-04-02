-- Migration: Add sender_key_requests table
-- Run this after postgres_e2ee.sql has already been applied.

CREATE TABLE IF NOT EXISTS public.sender_key_requests
(
    id                   BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    requester_member_id  BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    provider_member_id   BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    created_at           TIMESTAMP NOT NULL DEFAULT now(),
    fulfilled_at         TIMESTAMP,
    UNIQUE (requester_member_id, provider_member_id)
);
CREATE INDEX IF NOT EXISTS idx_sender_key_requests_provider
    ON public.sender_key_requests (provider_member_id)
    WHERE fulfilled_at IS NULL;
