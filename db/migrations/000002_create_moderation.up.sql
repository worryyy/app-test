CREATE TABLE IF NOT EXISTS moderation_reports (
    id BIGINT NOT NULL,
    reporter_root_user_id BIGINT NOT NULL,
    reporter_user_id BIGINT NOT NULL,
    target_type VARCHAR(32) NOT NULL,
    target_id VARCHAR(64) NOT NULL,
    target_root_user_id BIGINT NOT NULL,
    reason VARCHAR(64) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(24) NOT NULL,
    assignee_admin_id BIGINT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    withdrawn_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_moderation_reports_reporter (reporter_root_user_id, created_at DESC),
    KEY idx_moderation_reports_target_user (target_root_user_id, created_at DESC),
    KEY idx_moderation_reports_review (status, created_at ASC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS moderation_report_snapshots (
    id BIGINT NOT NULL,
    report_id BIGINT NOT NULL,
    payload JSON NOT NULL,
    created_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_moderation_snapshot_report (report_id),
    CONSTRAINT fk_moderation_snapshot_report FOREIGN KEY (report_id) REFERENCES moderation_reports(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS moderation_punishments (
    id BIGINT NOT NULL,
    root_user_id BIGINT NOT NULL,
    report_id BIGINT NULL,
    capability VARCHAR(24) NOT NULL,
    reason VARCHAR(255) NOT NULL,
    status VARCHAR(24) NOT NULL,
    starts_at DATETIME(3) NOT NULL,
    ends_at DATETIME(3) NULL,
    created_by BIGINT NOT NULL,
    created_at DATETIME(3) NOT NULL,
    revoked_by BIGINT NULL,
    revoked_at DATETIME(3) NULL,
    revoke_reason VARCHAR(255) NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_moderation_punishments_active (root_user_id, status, starts_at, ends_at),
    KEY idx_moderation_punishments_report (report_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS moderation_appeals (
    id BIGINT NOT NULL,
    punishment_id BIGINT NOT NULL,
    root_user_id BIGINT NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(24) NOT NULL,
    resolution VARCHAR(255) NOT NULL DEFAULT '',
    reviewed_by BIGINT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_moderation_appeal_punishment (punishment_id),
    KEY idx_moderation_appeals_review (status, created_at ASC),
    CONSTRAINT fk_moderation_appeal_punishment FOREIGN KEY (punishment_id) REFERENCES moderation_punishments(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS moderation_audit_logs (
    id BIGINT NOT NULL,
    report_id BIGINT NULL,
    punishment_id BIGINT NULL,
    appeal_id BIGINT NULL,
    admin_id BIGINT NOT NULL,
    action VARCHAR(32) NOT NULL,
    detail JSON NULL,
    created_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_moderation_audit_report (report_id, created_at ASC),
    KEY idx_moderation_audit_punishment (punishment_id, created_at ASC),
    KEY idx_moderation_audit_appeal (appeal_id, created_at ASC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
