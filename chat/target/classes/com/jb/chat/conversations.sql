CREATE TABLE conversations (
                               id VARCHAR(255) PRIMARY KEY COMMENT '会话ID (UUID或者雪花算法)',
                               type TINYINT NOT NULL DEFAULT 1 COMMENT '会话类型，1-单聊，2-群聊',
                               last_message_content TEXT COMMENT '最新一条消息的内容摘要，用于在会话列表展示',
                               last_message_sender_id VARCHAR(255) COMMENT '最新一条消息的发送者ID',
                               last_message_sent_at TIMESTAMP COMMENT '最新一条消息的发送时间',
                               created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                               updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) COMMENT='存储会话信息的表';

CREATE TABLE conversation_members (
                                      conversation_id VARCHAR(255) NOT NULL COMMENT '会话ID',
                                      user_id VARCHAR(255) NOT NULL COMMENT '用户ID',
                                      last_read_message_id BIGINT UNSIGNED COMMENT '该用户在此会话中已读的最后一条消息ID',
                                      unread_count INT UNSIGNED DEFAULT 0 COMMENT '未读消息数，每次发消息时更新',
                                      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                      PRIMARY KEY (conversation_id, user_id) COMMENT '联合主键',
                                      INDEX idx_user_conversation (user_id, conversation_id) COMMENT '极其重要的反向索引'
)   ENGINE = InnoDB
    COMMENT='存储会话成员关系的表';