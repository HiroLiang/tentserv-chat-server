CREATE TYPE friendship_status AS ENUM ('pending', 'accepted', 'blocked');

CREATE TABLE IF NOT EXISTS goat.public.user_friendships
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT REFERENCES users(id) ON DELETE CASCADE,
    friend_id  BIGINT REFERENCES users(id) ON DELETE CASCADE,
    status     friendship_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (user_id, friend_id)
);

CREATE INDEX IF NOT EXISTS idx_user_friendships_user_id ON goat.public.user_friendships(user_id);
CREATE INDEX IF NOT EXISTS idx_user_friendships_friend_id ON goat.public.user_friendships(friend_id);
