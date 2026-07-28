CREATE TABLE IF NOT EXISTS agent_conversations (
    session_id VARCHAR(64) NOT NULL,
    root_user_id BIGINT NOT NULL,
    creator_user_id BIGINT NOT NULL,
    last_actor_user_id BIGINT NOT NULL,
    title VARCHAR(128) NOT NULL DEFAULT '',
    last_user_preview VARCHAR(255) NOT NULL DEFAULT '',
    last_assistant_preview VARCHAR(255) NOT NULL DEFAULT '',
    last_request_id VARCHAR(64) NOT NULL DEFAULT '',
    last_trace_id VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (session_id),
    KEY idx_agent_conversations_root_updated (root_user_id, status, updated_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
