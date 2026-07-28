package academic

import (
	"context"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"

	filemodule "github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
)

var (
	ErrCourseNotFound   = bizerr.NotFound("课程不存在")
	ErrCourseDuplicated = bizerr.Biz("课程已存在")
	ErrReviewNotFound   = bizerr.NotFound("评价不存在")
	ErrMaterialNotFound = bizerr.NotFound("资料不存在")
)

type ProfileResolver interface {
	RootSchool(ctx context.Context, rootUserID int64) (string, error)
}
type CapabilityChecker interface {
	CheckCapability(ctx context.Context, userID, rootUserID int64, capability string) error
}

type FileStore interface {
	UploadAcademicDocument(ctx context.Context, stream multipart.File, header *multipart.FileHeader, userID string) (*filemodule.OriginalUpload, error)
	ReleaseReference(ctx context.Context, md5Value string) error
	OriginalURL(md5Value string) string
}

type Service struct {
	repo         *Repository
	profiles     ProfileResolver
	capabilities CapabilityChecker
	filter       sensitive.Filter
	files        FileStore
	logger       *zap.Logger
	now          func() time.Time
}

func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: NewRepository(db), logger: logger, now: time.Now}
}
func (s *Service) SetProfileResolver(value ProfileResolver)     { s.profiles = value }
func (s *Service) SetCapabilityChecker(value CapabilityChecker) { s.capabilities = value }
func (s *Service) SetSensitiveFilter(value sensitive.Filter)    { s.filter = value }
func (s *Service) SetFileStore(value FileStore)                 { s.files = value }

func (s *Service) CreateCourse(ctx context.Context, userID, rootUserID int64, req CreateCourseReq) (*CourseResponse, error) {
	if err := s.checkContent(ctx, userID, rootUserID); err != nil {
		return nil, err
	}
	if s.profiles == nil {
		return nil, bizerr.Internal("用户资料能力未配置")
	}
	school, err := s.profiles.RootSchool(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(school) == "" {
		return nil, bizerr.Biz("主账号未设置学校")
	}
	name, err := s.filterText(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	teacher, err := s.filterText(ctx, req.Teacher)
	if err != nil {
		return nil, err
	}
	description, err := s.filterText(ctx, req.Description)
	if err != nil {
		return nil, err
	}
	now := s.now()
	course := &Course{ID: snowflake.Generate().Int64(), School: strings.TrimSpace(school), Name: strings.TrimSpace(name), NormalizedName: normalizeName(name), Teacher: strings.TrimSpace(teacher), NormalizedTeacher: normalizeName(teacher), Description: description, Status: StatusNormal, CreatedByRootUserID: rootUserID, CreatedAt: now, UpdatedAt: now}
	if course.NormalizedName == "" {
		return nil, bizerr.Param("课程名不能为空")
	}
	existing, err := s.repo.FindCourseByIdentity(ctx, course.School, course.NormalizedName, course.NormalizedTeacher)
	if err != nil {
		return nil, bizerr.InternalWrap("查询课程失败", err)
	}
	if existing != nil {
		return nil, ErrCourseDuplicated
	}
	if err := s.repo.CreateCourse(ctx, course); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, ErrCourseDuplicated
		}
		return nil, bizerr.InternalWrap("创建课程失败", err)
	}
	response := courseResponse(*course, RatingStats{})
	return &response, nil
}

func (s *Service) SearchCourses(ctx context.Context, rootUserID int64, keyword string, page, size int) (*pagination.PageResult[CourseResponse], error) {
	if s.profiles == nil {
		return nil, bizerr.Internal("用户资料能力未配置")
	}
	school, err := s.profiles.RootSchool(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	page, size = normalizePage(page, size)
	list, total, err := s.repo.SearchCourses(ctx, strings.TrimSpace(school), strings.TrimSpace(keyword), page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("搜索课程失败", err)
	}
	items := make([]CourseResponse, 0, len(list))
	for _, item := range list {
		items = append(items, courseResponse(item, RatingStats{}))
	}
	return pagination.NewPageResult(items, total, page, size), nil
}

func (s *Service) CourseDetail(ctx context.Context, courseID string) (*CourseResponse, error) {
	idValue, err := parseID(courseID)
	if err != nil {
		return nil, err
	}
	course, err := s.repo.FindCourse(ctx, idValue)
	if err != nil {
		return nil, bizerr.InternalWrap("查询课程失败", err)
	}
	if course == nil || course.Status == StatusHidden {
		return nil, ErrCourseNotFound
	}
	if course.Status == StatusMerged && course.MergeTargetID != nil {
		course, err = s.repo.FindCourse(ctx, *course.MergeTargetID)
	}
	if err != nil || course == nil {
		return nil, ErrCourseNotFound
	}
	stats, err := s.repo.RatingStats(ctx, course.ID)
	if err != nil {
		return nil, bizerr.InternalWrap("查询评分统计失败", err)
	}
	response := courseResponse(*course, stats)
	return &response, nil
}

func (s *Service) ListReviews(ctx context.Context, courseID string, page, size int) (*pagination.PageResult[ReviewResponse], error) {
	idValue, err := parseID(courseID)
	if err != nil {
		return nil, err
	}
	page, size = normalizePage(page, size)
	list, total, err := s.repo.ListReviews(ctx, idValue, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询评价失败", err)
	}
	items := make([]ReviewResponse, 0, len(list))
	for _, item := range list {
		items = append(items, reviewResponse(item))
	}
	return pagination.NewPageResult(items, total, page, size), nil
}

func (s *Service) SaveReview(ctx context.Context, userID, rootUserID int64, courseID string, req ReviewReq) (*ReviewResponse, error) {
	if err := s.checkContent(ctx, userID, rootUserID); err != nil {
		return nil, err
	}
	idValue, err := parseID(courseID)
	if err != nil {
		return nil, err
	}
	course, err := s.repo.FindCourse(ctx, idValue)
	if err != nil || course == nil || course.Status != StatusNormal {
		return nil, ErrCourseNotFound
	}
	if !validRatings(req) {
		return nil, bizerr.Param("评分必须为1到5的整数")
	}
	content, err := s.filterText(ctx, req.Content)
	if err != nil {
		return nil, err
	}
	now := s.now()
	review := &Review{ID: snowflake.Generate().Int64(), CourseID: idValue, RootUserID: rootUserID, Semester: strings.TrimSpace(req.Semester), Content: content, OverallRating: req.OverallRating, DifficultyRating: req.DifficultyRating, WorkloadRating: req.WorkloadRating, GainRating: req.GainRating, Status: StatusNormal, CreatedAt: now, UpdatedAt: now}
	if review.Semester == "" {
		return nil, bizerr.Param("学期不能为空")
	}
	if err := s.repo.SaveReview(ctx, review); err != nil {
		return nil, bizerr.InternalWrap("保存评价失败", err)
	}
	response := reviewResponse(*review)
	return &response, nil
}

func (s *Service) DeleteReview(ctx context.Context, rootUserID int64, courseID string) error {
	idValue, err := parseID(courseID)
	if err != nil {
		return err
	}
	ok, err := s.repo.DeleteReview(ctx, idValue, rootUserID, s.now())
	if err != nil {
		return bizerr.InternalWrap("删除评价失败", err)
	}
	if !ok {
		return ErrReviewNotFound
	}
	return nil
}

func (s *Service) UploadMaterial(ctx context.Context, userID, rootUserID int64, courseID, semester, title, description string, stream multipart.File, header *multipart.FileHeader) (*MaterialResponse, error) {
	if err := s.checkContent(ctx, userID, rootUserID); err != nil {
		return nil, err
	}
	if s.files == nil {
		return nil, bizerr.Internal("文件能力未配置")
	}
	idValue, err := parseID(courseID)
	if err != nil {
		return nil, err
	}
	course, err := s.repo.FindCourse(ctx, idValue)
	if err != nil || course == nil || course.Status != StatusNormal {
		return nil, ErrCourseNotFound
	}
	semester, title, description = strings.TrimSpace(semester), strings.TrimSpace(title), strings.TrimSpace(description)
	if semester == "" || title == "" {
		return nil, bizerr.Param("学期和标题不能为空")
	}
	title, err = s.filterText(ctx, title)
	if err != nil {
		return nil, err
	}
	description, err = s.filterText(ctx, description)
	if err != nil {
		return nil, err
	}
	upload, err := s.files.UploadAcademicDocument(ctx, stream, header, strconv.FormatInt(rootUserID, 10))
	if err != nil {
		return nil, err
	}
	now := s.now()
	item := &Material{ID: snowflake.Generate().Int64(), CourseID: idValue, RootUserID: rootUserID, Semester: semester, Title: title, Description: description, OriginalName: upload.OriginalName, MIMEType: upload.MIME, SizeBytes: upload.Size, FileMD5: upload.MD5, Status: StatusNormal, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateMaterial(ctx, item); err != nil {
		_ = s.files.ReleaseReference(ctx, upload.MD5)
		return nil, bizerr.InternalWrap("保存资料失败", err)
	}
	response := materialResponse(*item)
	return &response, nil
}

func (s *Service) ListMaterials(ctx context.Context, courseID string, page, size int) (*pagination.PageResult[MaterialResponse], error) {
	idValue, err := parseID(courseID)
	if err != nil {
		return nil, err
	}
	page, size = normalizePage(page, size)
	list, total, err := s.repo.ListMaterials(ctx, "course_id = ? AND status = ?", []any{idValue, StatusNormal}, page, size)
	if err != nil {
		return nil, err
	}
	return materialPage(list, total, page, size), nil
}

func (s *Service) ListMyMaterials(ctx context.Context, rootUserID int64, page, size int) (*pagination.PageResult[MaterialResponse], error) {
	page, size = normalizePage(page, size)
	list, total, err := s.repo.ListMaterials(ctx, "root_user_id = ? AND status <> ?", []any{rootUserID, StatusDeleted}, page, size)
	if err != nil {
		return nil, err
	}
	return materialPage(list, total, page, size), nil
}

func (s *Service) MaterialDownloadURL(ctx context.Context, rootUserID int64, materialID string) (string, error) {
	idValue, err := parseID(materialID)
	if err != nil {
		return "", err
	}
	item, err := s.repo.FindMaterial(ctx, idValue)
	if err != nil {
		return "", err
	}
	if item == nil || (item.Status != StatusNormal && item.RootUserID != rootUserID) {
		return "", ErrMaterialNotFound
	}
	return s.files.OriginalURL(item.FileMD5), nil
}

func (s *Service) DeleteMaterial(ctx context.Context, rootUserID int64, materialID string) error {
	idValue, err := parseID(materialID)
	if err != nil {
		return err
	}
	item, err := s.repo.FindMaterial(ctx, idValue)
	if err != nil || item == nil || item.RootUserID != rootUserID {
		return ErrMaterialNotFound
	}
	ok, err := s.repo.UpdateMaterialStatus(ctx, idValue, &rootUserID, StatusDeleted, s.now())
	if err != nil || !ok {
		return ErrMaterialNotFound
	}
	if err := s.files.ReleaseReference(ctx, item.FileMD5); err != nil {
		s.logger.Warn("release material file failed", zap.Error(err), zap.String("materialID", materialID))
	}
	return nil
}

func (s *Service) MergeCourses(ctx context.Context, sourceID, targetID string) error {
	source, err := parseID(sourceID)
	if err != nil {
		return err
	}
	target, err := parseID(targetID)
	if err != nil {
		return err
	}
	if source == target {
		return bizerr.Param("不能合并同一课程")
	}
	if err := s.repo.MergeCourses(ctx, source, target, s.now()); err != nil {
		return bizerr.InternalWrap("合并课程失败", err)
	}
	return nil
}

func (s *Service) HideCourse(ctx context.Context, idValue string, hidden bool) error {
	return s.updateCourseStatus(ctx, idValue, hidden)
}
func (s *Service) HideReview(ctx context.Context, idValue string, hidden bool) error {
	idNum, err := parseID(idValue)
	if err != nil {
		return err
	}
	status := StatusNormal
	if hidden {
		status = StatusHidden
	}
	ok, err := s.repo.UpdateReviewStatus(ctx, idNum, status, s.now())
	if err != nil || !ok {
		return ErrReviewNotFound
	}
	return nil
}
func (s *Service) HideMaterial(ctx context.Context, idValue string, hidden bool) error {
	idNum, err := parseID(idValue)
	if err != nil {
		return err
	}
	status := StatusNormal
	if hidden {
		status = StatusHidden
	}
	ok, err := s.repo.UpdateMaterialStatus(ctx, idNum, nil, status, s.now())
	if err != nil || !ok {
		return ErrMaterialNotFound
	}
	return nil
}

func (s *Service) updateCourseStatus(ctx context.Context, raw string, hidden bool) error {
	idValue, err := parseID(raw)
	if err != nil {
		return err
	}
	status := StatusNormal
	if hidden {
		status = StatusHidden
	}
	ok, err := s.repo.UpdateCourseStatus(ctx, idValue, status, s.now())
	if err != nil || !ok {
		return ErrCourseNotFound
	}
	return nil
}

func (s *Service) ReportTarget(ctx context.Context, targetType, targetID string) (int64, any, error) {
	idValue, err := parseID(targetID)
	if err != nil {
		return 0, nil, err
	}
	switch targetType {
	case "courseReview":
		item, err := s.repo.FindReview(ctx, idValue)
		if err != nil || item == nil {
			return 0, nil, err
		}
		return item.RootUserID, item, nil
	case "material":
		item, err := s.repo.FindMaterial(ctx, idValue)
		if err != nil || item == nil {
			return 0, nil, err
		}
		return item.RootUserID, item, nil
	default:
		return 0, nil, ErrCourseNotFound
	}
}
func (s *Service) HideReportTarget(ctx context.Context, targetType, targetID string) error {
	if targetType == "courseReview" {
		return s.HideReview(ctx, targetID, true)
	}
	if targetType == "material" {
		return s.HideMaterial(ctx, targetID, true)
	}
	return ErrCourseNotFound
}

func (s *Service) filterText(ctx context.Context, value string) (string, error) {
	if s.filter == nil {
		return strings.TrimSpace(value), nil
	}
	filtered, err := s.filter.FilterText(ctx, value)
	if err != nil {
		return "", bizerr.InternalWrap("敏感词过滤失败", err)
	}
	return strings.TrimSpace(filtered), nil
}
func (s *Service) checkContent(ctx context.Context, userID, rootUserID int64) error {
	if s.capabilities == nil {
		return nil
	}
	return s.capabilities.CheckCapability(ctx, userID, rootUserID, "content")
}
func normalizeName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(norm.NFKC.String(value)), " "))
}
func validRatings(req ReviewReq) bool {
	return between(req.OverallRating) && between(req.DifficultyRating) && between(req.WorkloadRating) && between(req.GainRating)
}
func between(value int) bool { return value >= 1 && value <= 5 }
func parseID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, bizerr.Param("ID格式错误")
	}
	return value, nil
}
func normalizePage(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
