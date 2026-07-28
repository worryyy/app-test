package academic

import "time"

const (
	StatusNormal  = "normal"
	StatusHidden  = "hidden"
	StatusDeleted = "deleted"
	StatusMerged  = "merged"
)

type Course struct {
	ID                  int64     `gorm:"column:id;primaryKey"`
	School              string    `gorm:"column:school"`
	Name                string    `gorm:"column:name"`
	NormalizedName      string    `gorm:"column:normalized_name"`
	Teacher             string    `gorm:"column:teacher"`
	NormalizedTeacher   string    `gorm:"column:normalized_teacher"`
	Description         string    `gorm:"column:description"`
	Status              string    `gorm:"column:status"`
	MergeTargetID       *int64    `gorm:"column:merge_target_id"`
	CreatedByRootUserID int64     `gorm:"column:created_by_root_user_id"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (Course) TableName() string { return "academic_courses" }

type Review struct {
	ID               int64     `gorm:"column:id;primaryKey"`
	CourseID         int64     `gorm:"column:course_id"`
	RootUserID       int64     `gorm:"column:root_user_id"`
	Semester         string    `gorm:"column:semester"`
	Content          string    `gorm:"column:content"`
	OverallRating    int       `gorm:"column:overall_rating"`
	DifficultyRating int       `gorm:"column:difficulty_rating"`
	WorkloadRating   int       `gorm:"column:workload_rating"`
	GainRating       int       `gorm:"column:gain_rating"`
	Status           string    `gorm:"column:status"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (Review) TableName() string { return "academic_reviews" }

type Material struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	CourseID     int64     `gorm:"column:course_id"`
	RootUserID   int64     `gorm:"column:root_user_id"`
	Semester     string    `gorm:"column:semester"`
	Title        string    `gorm:"column:title"`
	Description  string    `gorm:"column:description"`
	OriginalName string    `gorm:"column:original_name"`
	MIMEType     string    `gorm:"column:mime_type"`
	SizeBytes    int64     `gorm:"column:size_bytes"`
	FileMD5      string    `gorm:"column:file_md5"`
	Status       string    `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (Material) TableName() string { return "academic_materials" }

type RatingStats struct {
	Count      int64   `json:"count"`
	Overall    float64 `json:"overall"`
	Difficulty float64 `json:"difficulty"`
	Workload   float64 `json:"workload"`
	Gain       float64 `json:"gain"`
}
