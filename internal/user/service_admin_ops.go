package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

const blacklistDocID = "global_blacklist"

func (s *Service) CreateUser(ctx context.Context, u *User) error {
	if u == nil {
		return result.ErrParam
	}
	if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *Service) AddAdmin(ctx context.Context, userID int64, username, password string, power *int) error {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return result.NewBizError(result.CodeFail, "关联用户不存在")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&Admin{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return fmt.Errorf("count admin by username: %w", err)
	}
	if count > 0 {
		return result.NewBizError(result.CodeFail, "用户名重复")
	}
	if err := s.db.WithContext(ctx).Model(&Admin{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return fmt.Errorf("count admin by user id: %w", err)
	}
	if count > 0 {
		return result.NewBizError(result.CodeFail, "管理员已存在")
	}

	adminPower := 0
	if power != nil {
		adminPower = *power
	}
	admin := Admin{
		UserID:   userID,
		Username: username,
		Password: md5Hex(password),
		Power:    resolveAdminPower(adminPower),
	}
	if err := s.db.WithContext(ctx).Create(&admin).Error; err != nil {
		return result.NewBizError(result.CodeFail, "添加失败，请重试")
	}
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Delete(&User{}, id).Error; err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *Service) EditAdminUser(ctx context.Context, userID, operatorID int64, req AdminEditUserReq) error {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return result.NewBizError(result.CodeFail, "不存在")
	}

	updates := map[string]interface{}{}
	if req.StuNum != "" {
		updates["stu_num"] = req.StuNum
	}
	if req.StuCla != "" {
		updates["stuCla"] = req.StuCla
	}
	if req.StuName != "" {
		updates["stuName"] = req.StuName
	}
	if req.StuIsCheck != nil {
		updates["stu_is_check"] = *req.StuIsCheck
	}
	if req.Power != nil {
		updates["power"] = *req.Power
	}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if operatorID > 0 {
		updates["updatedBy"] = operatorID
	}
	if len(updates) == 0 {
		return result.NewBizError(result.CodeFail, "更新失败")
	}

	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return fmt.Errorf("edit admin user: %w", err)
	}

	if s.producer != nil {
		msg := mq.TopicUserUpdateMsg{
			UserID:      strconv.FormatInt(userID, 10),
			NickName:    req.Nickname,
			Avatar:      req.Avatar,
			AccountType: mapAccountType(user.AccountType),
		}
		if err := s.producer.SendUpdateTopicUser(ctx, msg); err != nil {
			s.logger.Warn("send admin topic user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
		if err := s.producer.SendUpdateCommentUser(ctx, mq.CommentUserUpdateMsg(msg)); err != nil {
			s.logger.Warn("send admin comment user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
	}
	return nil
}

func (s *Service) ListUsers(ctx context.Context, page, size int, name string) (*result.PageResult[User], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = s.defaultPageSize()
	}
	q := s.db.WithContext(ctx).Model(&User{})
	if name != "" {
		q = q.Where("nickname = ?", name)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	var list []User
	if err := q.Offset((page - 1) * size).Limit(size).Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return result.NewPage(s.sanitizeUsers(list), total, page, size), nil
}

func (s *Service) ClearAuthentication(ctx context.Context, userID int64) error {
	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"stu_is_check": false,
		"stuName":    "",
		"stuCla":     "",
		"stu_num":     "",
	}).Error; err != nil {
		return fmt.Errorf("clear authentication: %w", err)
	}
	return nil
}

func (s *Service) AddBlackList(ctx context.Context, identifiers []string) error {
	rootIDs, err := s.resolveBlacklistRootIDs(ctx, identifiers)
	if err != nil {
		return err
	}
	if len(rootIDs) == 0 {
		return result.ErrParam
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"updated_time": now,
		},
		"$setOnInsert": bson.M{
			"_id":          blacklistDocID,
			"created_time": now,
		},
		"$addToSet": bson.M{
			"blocked_user_ids": bson.M{"$each": rootIDs},
		},
	}
	if _, err := s.blacklistColl().UpdateByID(ctx, blacklistDocID, update, options.Update().SetUpsert(true)); err != nil {
		return fmt.Errorf("upsert blacklist doc: %w", err)
	}
	if s.redis != nil {
		members := make([]interface{}, 0, len(rootIDs))
		for _, rootID := range rootIDs {
			members = append(members, rootID)
		}
		if err := s.redis.SAdd(ctx, rediskey.GlobalBlacklist, members...).Err(); err != nil {
			return fmt.Errorf("add blacklist users: %w", err)
		}
	}
	return nil
}

func (s *Service) DelBlackList(ctx context.Context, identifiers []string) error {
	rootIDs, err := s.resolveBlacklistRootIDs(ctx, identifiers)
	if err != nil {
		return err
	}
	if len(rootIDs) == 0 {
		return result.ErrParam
	}

	if _, err := s.blacklistColl().UpdateByID(
		ctx,
		blacklistDocID,
		bson.M{
			"$set": bson.M{"updated_time": time.Now()},
			"$pull": bson.M{
				"blocked_user_ids": bson.M{"$in": rootIDs},
			},
		},
	); err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("remove blacklist from mongo: %w", err)
	}
	if s.redis != nil {
		members := make([]interface{}, 0, len(rootIDs))
		for _, rootID := range rootIDs {
			members = append(members, rootID)
		}
		if err := s.redis.SRem(ctx, rediskey.GlobalBlacklist, members...).Err(); err != nil {
			return fmt.Errorf("remove blacklist users: %w", err)
		}
	}
	return nil
}

func (s *Service) ListBlackList(ctx context.Context) ([]User, error) {
	rootIDs, err := s.loadBlacklistRootIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(rootIDs) == 0 {
		return []User{}, nil
	}
	ids := make([]int64, 0, len(rootIDs))
	for _, m := range rootIDs {
		if v, convErr := strconv.ParseInt(m, 10, 64); convErr == nil {
			ids = append(ids, v)
		}
	}
	var users []User
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("query blacklist users: %w", err)
	}
	byID := make(map[int64]User, len(users))
	for _, u := range users {
		u.StuPwd = ""
		byID[u.ID] = u
	}
	ordered := make([]User, 0, len(ids))
	for _, id := range ids {
		if u, ok := byID[id]; ok {
			ordered = append(ordered, u)
		}
	}
	return ordered, nil
}

func (s *Service) ListOfficialCertifications(
	ctx context.Context,
	page, size int,
	status string,
) (*result.CusPage[OfficialCertificationListItem], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	coll := s.mongoDB.Collection("campus_official_certification")
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count certifications: %w", err)
	}
	cur, err := coll.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.M{"_id": -1}).SetSkip(int64((page-1)*size)).SetLimit(int64(size)),
	)
	if err != nil {
		return nil, fmt.Errorf("find certifications: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close certification cursor failed", zap.Error(closeErr))
		}
	}()

	var list []OfficialCertification
	if err := cur.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode certifications: %w", err)
	}
	items := make([]OfficialCertificationListItem, 0, len(list))
	for _, item := range list {
		items = append(items, OfficialCertificationListItem{
			ID:                item.ID.Hex(),
			AvatarURL:         item.AvatarURL,
			FullName:          item.FullName,
			ShortName:         item.ShortName,
			Nature:            item.Nature,
			Introduction:      item.Introduction,
			ResponsiblePerson: item.ResponsiblePerson,
			WechatAccount:     item.WechatAccount,
			LoginAccount:      item.LoginAccount,
			Status:            item.Status,
			RejectReason:      item.RejectReason,
			ReviewedBy:        item.ReviewedBy,
			ReviewedAt:        item.ReviewedAt,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		})
	}
	return result.NewCusPage(items, total, page, size), nil
}

func (s *Service) ReviewCertification(ctx context.Context, reviewerID int64, req CertReviewReq) error {
	oid, err := primitive.ObjectIDFromHex(req.CertificationID)
	if err != nil {
		return fmt.Errorf("invalid cert id: %w", err)
	}

	var cert OfficialCertification
	if err := s.mongoDB.Collection("campus_official_certification").FindOne(ctx, bson.M{"_id": oid}).Decode(&cert); err != nil {
		if err == mongo.ErrNoDocuments {
			return result.NewBizError(result.CodeFail, "认证申请不存在")
		}
		return fmt.Errorf("find certification: %w", err)
	}

	if cert.Status != certificationStatusPending {
		return result.NewBizError(result.CodeFail, "该申请已审核，无法重复审核")
	}

	now := time.Now()
	switch req.Action {
	case certificationStatusApproved:
		var count int64
		if err := s.db.WithContext(ctx).Model(&User{}).Where("stu_num = ?", cert.LoginAccount).Count(&count).Error; err != nil {
			return fmt.Errorf("check official login account in users: %w", err)
		}
		if count > 0 {
			return result.NewBizError(result.CodeFail, "该登录账号已被使用，无法创建用户")
		}

		avatar := cert.AvatarURL
		if avatar == "" {
			avatar = s.pickDefaultAvatar()
		}
		officialUser := &User{
			Nickname:    cert.ShortName,
			Avatar:      avatar,
			OpenID:      "official:" + cert.LoginAccount,
			StuNum:      cert.LoginAccount,
			StuPwd:      cert.LoginPassword,
			StuName:     cert.ResponsiblePerson,
			StuIsCheck:  true,
			Power:       0,
			CreatedBy:   reviewerID,
			UpdatedBy:   reviewerID,
			Tag:         req.Tag,
			AccountType: accountTypeBase,
		}
		if err := s.db.WithContext(ctx).Create(officialUser).Error; err != nil {
			return result.NewBizError(result.CodeFail, "创建用户失败，请重试")
		}
		if err := s.mongoDB.Collection("campus_official_certification").FindOneAndUpdate(
			ctx,
			bson.M{"_id": oid},
			bson.M{"$set": bson.M{
				"status":     certificationStatusApproved,
				"reviewedBy": reviewerID,
				"reviewedAt": now,
				"updated_at":  now,
			}},
		).Err(); err != nil {
			return fmt.Errorf("update certification approved status: %w", err)
		}
		return nil
	case certificationStatusRejected:
		if strings.TrimSpace(req.RejectReason) == "" {
			return result.NewBizError(result.CodeFail, "拒绝原因不能为空")
		}
		if err := s.mongoDB.Collection("campus_official_certification").FindOneAndUpdate(
			ctx,
			bson.M{"_id": oid},
			bson.M{"$set": bson.M{
				"status":       certificationStatusRejected,
				"rejectReason": req.RejectReason,
				"reviewedBy":   reviewerID,
				"reviewedAt":   now,
				"updated_at":    now,
			}},
		).Err(); err != nil {
			return fmt.Errorf("update certification rejected status: %w", err)
		}
		return nil
	default:
		return result.NewBizError(result.CodeFail, "审核操作非法，只能选择 APPROVED 或 REJECTED")
	}
}

func (s *Service) GetCourseFileByKey(ctx context.Context, key string) (*CourseFile, error) {
	if strings.TrimSpace(key) == "" {
		return nil, result.ErrParam
	}

	var course CourseFile
	if err := s.mongoDB.Collection("campus_course").FindOne(ctx, bson.M{"key": key}).Decode(&course); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, result.NewBizStatusError(404, result.CodeFail, "获取课表失败,请联系管理员")
		}
		return nil, fmt.Errorf("find course by key: %w", err)
	}
	return &course, nil
}

type CourseFile struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Key      string             `bson:"key"`
	FilePath string             `bson:"filePath"`
	Val      []byte             `bson:"val"`
}

func (s *Service) blacklistColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_user_blacklist")
}

func (s *Service) loadBlacklistRootIDs(ctx context.Context) ([]string, error) {
	if s.redis != nil {
		members, err := s.redis.SMembers(ctx, rediskey.GlobalBlacklist).Result()
		if err == nil && len(members) > 0 {
			return members, nil
		}
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("list blacklist members: %w", err)
		}
	}

	var doc UserBlacklist
	err := s.blacklistColl().FindOne(ctx, bson.M{"_id": blacklistDocID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []string{}, nil
		}
		return nil, fmt.Errorf("find blacklist doc: %w", err)
	}

	rootIDs, err := s.resolveBlacklistRootIDs(ctx, doc.BlockedUserIDs)
	if err != nil {
		return nil, err
	}
	if len(rootIDs) == 0 {
		return []string{}, nil
	}

	if s.redis != nil {
		members := make([]interface{}, 0, len(rootIDs))
		for _, rootID := range rootIDs {
			members = append(members, rootID)
		}
		if err := s.redis.SAdd(ctx, rediskey.GlobalBlacklist, members...).Err(); err != nil {
			return nil, fmt.Errorf("restore blacklist cache: %w", err)
		}
	}

	if !sameStringSlice(doc.BlockedUserIDs, rootIDs) {
		_, updateErr := s.blacklistColl().UpdateByID(ctx, blacklistDocID, bson.M{
			"$set": bson.M{
				"blocked_user_ids": rootIDs,
				"updated_time":     time.Now(),
			},
		})
		if updateErr != nil {
			return nil, fmt.Errorf("migrate blacklist doc: %w", updateErr)
		}
	}
	return rootIDs, nil
}

func (s *Service) resolveBlacklistRootIDs(ctx context.Context, identifiers []string) ([]string, error) {
	seen := make(map[string]struct{})
	rootIDs := make([]string, 0, len(identifiers))
	for _, raw := range identifiers {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		var target *User
		if id, convErr := strconv.ParseInt(raw, 10, 64); convErr == nil && id > 0 {
			target, _ = s.GetByID(ctx, id)
		}
		if target == nil {
			var err error
			target, err = s.GetByOpenID(ctx, raw)
			if err != nil {
				return nil, err
			}
		}
		if target == nil {
			continue
		}

		rootID := strconv.FormatInt(rootUserID(target), 10)
		if _, ok := seen[rootID]; ok {
			continue
		}
		seen[rootID] = struct{}{}
		rootIDs = append(rootIDs, rootID)
	}
	return rootIDs, nil
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
