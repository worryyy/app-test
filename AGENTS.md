# AGENTS.md
### 返回响应体代码规范 
## service 层怎么写
参数错误
func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
	if id <= 0 {
		return nil, bizerr.Param("用户ID不合法")
	}

	// ...
	return &User{}, nil
}
查无此人

如果你用的是 gorm，可以这样：

func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizerr.NotFound("用户不存在")
		}
		return nil, bizerr.InternalWrap("查询用户失败", err)
	}
	return user, nil
}
业务错误
func (s *OrderService) Pay(ctx context.Context, orderID int64) error {
	ok := checkBalance()
	if !ok {
		return bizerr.Biz("余额不足")
	}
	return nil
}
系统错误
func (s *UserService) Create(ctx context.Context, req CreateUserReq) error {
	if err := s.repo.Create(ctx, req); err != nil {
		return bizerr.InternalWrap("创建用户失败", err)
	}
	return nil
}
## controller层怎么写 
func (h *UserHandler) GetUser(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		responses.ParamErr.RespMessage(ctx, "id格式错误")
		return
	}

	user, err := h.userService.GetUser(ctx, id)
	if err != nil {
		responses.Fail(ctx, err)
		return
	}

	responses.Success.RespData(ctx, user)
}
带自定义成功消息
func (h *UserHandler) Login(ctx *gin.Context) {
	user, err := h.userService.Login(ctx)
	if err != nil {
		responses.Fail(ctx, err)
		return
	}

	responses.Success.RespMessageData(ctx, "登录成功", user)
}
## err不为空补上日志
if err != nil {
	log.Errorf("get user failed: %v", err)
	responses.Fail(ctx, err)
	return
}
## service 层

统一返回：

bizerr.Param(...)
bizerr.Biz(...)
bizerr.NotFound(...)
bizerr.InternalWrap(...)
## controller 层

统一返回：

responses.Success.Resp(ctx)
responses.Success.RespData(ctx, data)
responses.Success.RespMessage(ctx, msg)
responses.Success.RespMessageData(ctx, msg, data)

responses.Fail(ctx, err)

## 什么时候使用warp 什么时候不使用 
Param / Biz / NotFound：通常直接 New
Internal / 外部依赖失败：通常 Wrap