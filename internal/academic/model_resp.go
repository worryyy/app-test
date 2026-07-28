package academic

import (
	"strconv"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
)

type CourseResponse struct {
	ID          string      `json:"id"`
	School      string      `json:"school"`
	Name        string      `json:"name"`
	Teacher     string      `json:"teacher"`
	Description string      `json:"description"`
	Status      string      `json:"status"`
	Stats       RatingStats `json:"stats"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
}
type ReviewResponse struct {
	ID               string `json:"id"`
	CourseID         string `json:"courseId"`
	Semester         string `json:"semester"`
	Content          string `json:"content"`
	OverallRating    int    `json:"overallRating"`
	DifficultyRating int    `json:"difficultyRating"`
	WorkloadRating   int    `json:"workloadRating"`
	GainRating       int    `json:"gainRating"`
	UpdatedAt        string `json:"updatedAt"`
}
type MaterialResponse struct {
	ID           string `json:"id"`
	CourseID     string `json:"courseId"`
	Semester     string `json:"semester"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	OriginalName string `json:"originalName"`
	MIMEType     string `json:"mimeType"`
	SizeBytes    int64  `json:"sizeBytes"`
	MD5          string `json:"md5"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
}

func courseResponse(item Course, stats RatingStats) CourseResponse {
	return CourseResponse{ID: strconv.FormatInt(item.ID, 10), School: item.School, Name: item.Name, Teacher: item.Teacher, Description: item.Description, Status: item.Status, Stats: stats, CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt)}
}
func reviewResponse(item Review) ReviewResponse {
	return ReviewResponse{ID: strconv.FormatInt(item.ID, 10), CourseID: strconv.FormatInt(item.CourseID, 10), Semester: item.Semester, Content: item.Content, OverallRating: item.OverallRating, DifficultyRating: item.DifficultyRating, WorkloadRating: item.WorkloadRating, GainRating: item.GainRating, UpdatedAt: formatTime(item.UpdatedAt)}
}
func materialResponse(item Material) MaterialResponse {
	return MaterialResponse{ID: strconv.FormatInt(item.ID, 10), CourseID: strconv.FormatInt(item.CourseID, 10), Semester: item.Semester, Title: item.Title, Description: item.Description, OriginalName: item.OriginalName, MIMEType: item.MIMEType, SizeBytes: item.SizeBytes, MD5: item.FileMD5, Status: item.Status, CreatedAt: formatTime(item.CreatedAt)}
}
func materialPage(list []Material, total int64, page, size int) *pagination.PageResult[MaterialResponse] {
	items := make([]MaterialResponse, 0, len(list))
	for _, item := range list {
		items = append(items, materialResponse(item))
	}
	return pagination.NewPageResult(items, total, page, size)
}
func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return value.In(loc).Format(time.RFC3339)
}
