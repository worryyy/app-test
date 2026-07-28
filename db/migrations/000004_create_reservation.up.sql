CREATE TABLE IF NOT EXISTS reservation_venues (
    id BIGINT NOT NULL, name VARCHAR(128) NOT NULL, description TEXT NOT NULL,
    status VARCHAR(24) NOT NULL, advance_days INT NOT NULL DEFAULT 7,
    slot_minutes INT NOT NULL DEFAULT 60, daily_limit INT NOT NULL DEFAULT 2,
    cancel_before_minutes INT NOT NULL DEFAULT 120,
    created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id), KEY idx_reservation_venues_status (status, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS reservation_resources (
    id BIGINT NOT NULL, venue_id BIGINT NOT NULL, name VARCHAR(128) NOT NULL,
    capacity INT NOT NULL DEFAULT 1, status VARCHAR(24) NOT NULL,
    created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id), KEY idx_reservation_resources_venue (venue_id, status),
    CONSTRAINT fk_reservation_resource_venue FOREIGN KEY (venue_id) REFERENCES reservation_venues(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS reservation_weekly_rules (
    id BIGINT NOT NULL, resource_id BIGINT NOT NULL, weekday TINYINT NOT NULL,
    start_minute SMALLINT NOT NULL, end_minute SMALLINT NOT NULL, status VARCHAR(24) NOT NULL,
    created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id), KEY idx_reservation_rules_resource_day (resource_id, weekday, status),
    CONSTRAINT fk_reservation_rule_resource FOREIGN KEY (resource_id) REFERENCES reservation_resources(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS reservation_closures (
    id BIGINT NOT NULL, venue_id BIGINT NULL, resource_id BIGINT NULL,
    start_at DATETIME(3) NOT NULL, end_at DATETIME(3) NOT NULL, reason VARCHAR(255) NOT NULL,
    created_by BIGINT NOT NULL, created_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id), KEY idx_reservation_closures_time (venue_id, resource_id, start_at, end_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS reservation_slots (
    id BIGINT NOT NULL, resource_id BIGINT NOT NULL, start_at DATETIME(3) NOT NULL,
    end_at DATETIME(3) NOT NULL, capacity INT NOT NULL, reserved_count INT NOT NULL DEFAULT 0,
    status VARCHAR(24) NOT NULL, created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id), UNIQUE KEY uk_reservation_slot (resource_id, start_at, end_at),
    KEY idx_reservation_slots_availability (resource_id, start_at, status),
    CONSTRAINT fk_reservation_slot_resource FOREIGN KEY (resource_id) REFERENCES reservation_resources(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS reservation_user_day_locks (
    id BIGINT NOT NULL, root_user_id BIGINT NOT NULL, reservation_date DATE NOT NULL,
    PRIMARY KEY (id), UNIQUE KEY uk_reservation_user_day (root_user_id, reservation_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS reservations (
    id BIGINT NOT NULL, root_user_id BIGINT NOT NULL, user_id BIGINT NOT NULL, slot_id BIGINT NOT NULL,
    status VARCHAR(24) NOT NULL, checkin_code VARCHAR(32) NOT NULL,
    checked_at DATETIME(3) NULL, cancelled_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id), UNIQUE KEY uk_reservation_checkin_code (checkin_code),
    KEY idx_reservations_owner (root_user_id, created_at DESC),
    KEY idx_reservations_slot_status (slot_id, status),
    CONSTRAINT fk_reservation_slot FOREIGN KEY (slot_id) REFERENCES reservation_slots(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
