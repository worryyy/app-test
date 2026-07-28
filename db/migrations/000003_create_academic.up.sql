CREATE TABLE IF NOT EXISTS academic_courses (
    id BIGINT NOT NULL,
    school VARCHAR(128) NOT NULL,
    name VARCHAR(128) NOT NULL,
    normalized_name VARCHAR(128) NOT NULL,
    teacher VARCHAR(128) NOT NULL DEFAULT '',
    normalized_teacher VARCHAR(128) NOT NULL DEFAULT '',
    description TEXT NOT NULL,
    status VARCHAR(24) NOT NULL,
    merge_target_id BIGINT NULL,
    created_by_root_user_id BIGINT NOT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_academic_course_identity (school, normalized_name, normalized_teacher),
    KEY idx_academic_courses_search (school, status, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS academic_reviews (
    id BIGINT NOT NULL,
    course_id BIGINT NOT NULL,
    root_user_id BIGINT NOT NULL,
    semester VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    overall_rating TINYINT UNSIGNED NOT NULL,
    difficulty_rating TINYINT UNSIGNED NOT NULL,
    workload_rating TINYINT UNSIGNED NOT NULL,
    gain_rating TINYINT UNSIGNED NOT NULL,
    status VARCHAR(24) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_academic_review_owner (course_id, root_user_id),
    KEY idx_academic_reviews_list (course_id, status, updated_at DESC),
    CONSTRAINT fk_academic_review_course FOREIGN KEY (course_id) REFERENCES academic_courses(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS academic_materials (
    id BIGINT NOT NULL,
    course_id BIGINT NOT NULL,
    root_user_id BIGINT NOT NULL,
    semester VARCHAR(32) NOT NULL,
    title VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    size_bytes BIGINT NOT NULL,
    file_md5 CHAR(32) NOT NULL,
    status VARCHAR(24) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_academic_materials_course (course_id, status, created_at DESC),
    KEY idx_academic_materials_owner (root_user_id, created_at DESC),
    CONSTRAINT fk_academic_material_course FOREIGN KEY (course_id) REFERENCES academic_courses(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
