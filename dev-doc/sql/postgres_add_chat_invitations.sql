CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'rejected');

CREATE TABLE IF NOT EXISTS public.chat_invitations
(
    id         BIGSERIAL PRIMARY KEY,
    room_id    BIGINT            NOT NULL REFERENCES public.chat_rooms (id),
    inviter_id BIGINT            NOT NULL REFERENCES public.participants (id),
    invitee_id BIGINT            NOT NULL REFERENCES public.participants (id),
    status     invitation_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ       NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ       NOT NULL DEFAULT now()
);

CREATE INDEX idx_chat_invitation_room_id_invite_id ON public.chat_invitations (room_id, invitee_id);
