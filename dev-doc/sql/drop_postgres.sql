---- Drop Indexes Query ----

-- Participant
DROP INDEX IF EXISTS idx_participants_user;
DROP INDEX IF EXISTS idx_participants_agent;
DROP INDEX IF EXISTS idx_participants_system;

-- Chat group members
DROP INDEX IF EXISTS idx_chat_group_members_participant;

-- Chat records
DROP INDEX IF EXISTS idx_chat_records_group_id;
DROP INDEX IF EXISTS idx_chat_records_group_created;
DROP INDEX IF EXISTS idx_chat_records_sender;

---- Drop Tables Query ----

-- Chats
DROP TABLE IF EXISTS goat.public.chat_record CASCADE;
DROP TABLE IF EXISTS goat.public.chat_group_members CASCADE;
DROP TABLE IF EXISTS goat.public.participants CASCADE;
DROP TABLE IF EXISTS goat.public.chat_groups CASCADE;

-- Agents
DROP TABLE IF EXISTS goat.public.agents CASCADE;

-- Users
DROP TABLE IF EXISTS goat.public.users_roles CASCADE;
DROP TABLE IF EXISTS goat.public.roles CASCADE;
DROP TABLE IF EXISTS goat.public.users CASCADE;

---- Drop Enums Query ----

-- Users
DROP TYPE IF EXISTS user_status;

-- Agents
DROP TYPE IF EXISTS agent_status;
DROP TYPE IF EXISTS agent_types;
DROP TYPE IF EXISTS agent_engine;

-- Chats
DROP TYPE IF EXISTS chat_group_type;
DROP TYPE IF EXISTS participant_type;
DROP TYPE IF EXISTS chat_message_type;
DROP TYPE IF EXISTS chat_member_role;
