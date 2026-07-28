package academic

type CreateCourseReq struct {
	Name        string `json:"name" binding:"required,max=128"`
	Teacher     string `json:"teacher" binding:"max=128"`
	Description string `json:"description" binding:"max=2000"`
}

type ReviewReq struct {
	Semester         string `json:"semester" binding:"required,max=32"`
	Content          string `json:"content" binding:"max=4000"`
	OverallRating    int    `json:"overallRating" binding:"required,min=1,max=5"`
	DifficultyRating int    `json:"difficultyRating" binding:"required,min=1,max=5"`
	WorkloadRating   int    `json:"workloadRating" binding:"required,min=1,max=5"`
	GainRating       int    `json:"gainRating" binding:"required,min=1,max=5"`
}

type MergeCourseReq struct {
	TargetCourseID string `json:"targetCourseId" binding:"required"`
}
type HideReq struct {
	Hidden bool `json:"hidden"`
}
