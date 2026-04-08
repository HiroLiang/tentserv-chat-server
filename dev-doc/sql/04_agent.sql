-- ============================================================
-- [EN] File: 04_agent.sql  |  Execution order: 4 of 10
--      Creates the AI agent layer: agent_status, agent_types,
--      and agent_engine enums, plus the agents table.
--
-- [中] 檔案：04_agent.sql  |  執行順序：第 4 支（共 10 支）
--      建立 AI Agent 層：agent_status、agent_types、agent_engine
--      列舉，以及 agents 資料表。
--
-- [日] ファイル：04_agent.sql  |  実行順：4/10
--      AI エージェント層を作成する：agent_status、agent_types、
--      agent_engine 列挙と agents テーブル。
--
-- Dependencies | 依賴 | 依存: 01_account.sql (users)
-- Creates      | 建立 | 作成:
--   TYPE  agent_status, agent_types, agent_engine
--   TABLE agents
-- ============================================================


-- ============================================================
-- SECTION 1: ENUM TYPES
-- [EN] Agent operational status, deployment type, and inference engine.
-- [中] Agent 運行狀態、部署類型及推理引擎的列舉定義。
-- [日] エージェントの動作ステータス、デプロイタイプ、推論エンジンを定義する。
-- ============================================================

-- Agent operational status | Agent 運行狀態 | エージェント動作ステータス
CREATE TYPE agent_status AS ENUM ('available', 'maintaining', 'discontinued', 'error');

-- Agent deployment type | Agent 部署類型 | エージェントデプロイタイプ
CREATE TYPE agent_types AS ENUM ('local', 'remote', 'hybrid');

-- Inference engine backend | 推理引擎後端 | 推論エンジンバックエンド
CREATE TYPE agent_engine AS ENUM ('gguf', 'onnx', 'api', 'cloud', 'mlc', 'webGPU');


-- ============================================================
-- SECTION 2: TABLES
-- [EN] agents stores registered AI agents. created_by / updated_by
--      reference users for ownership and audit tracking.
-- [中] agents 記錄已登錄的 AI Agent。created_by / updated_by
--      參照 users 以追蹤擁有者與稽核紀錄。
-- [日] agents は登録済み AI エージェントを格納する。
--      created_by / updated_by は所有者と監査追跡のために users を参照する。
-- ============================================================

-- AI Agents | AI Agent | AI エージェント
CREATE TABLE IF NOT EXISTS public.agents
(
    id         BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name       TEXT         NOT NULL,
    type       agent_types  NOT NULL,
    status     agent_status NOT NULL DEFAULT 'maintaining',
    engine     agent_engine NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT now(),
    created_by BIGINT REFERENCES users (id) ON DELETE CASCADE,
    updated_at TIMESTAMP    NOT NULL DEFAULT now(),
    updated_by BIGINT REFERENCES users (id) ON DELETE CASCADE
);
