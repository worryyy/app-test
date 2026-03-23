# 数据模型定义

> 所有模型的表名、字段名、集合名必须与 Java 版完全一致，以确保直接接管现有数据库。
>
> 每个 struct 定义在所属域的 `model.go` 中，一个 struct 同时携带 `gorm:`/`bson:` 和 `json:` tag，不搞 VO/DTO/PO 区分。
>
> **两个服务共享所有 model**——model.go 是域包的共享基础。

---

## 1. MySQL 实体（14 张表）

### 1.1 campus_user — `internal/user/model.go`

```go
type User struct {
    ID           int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    OpenID       string         `gorm:"column:openId" json:"openId"`
    Nickname     string         `gorm:"column:nickname" json:"nickname"`
    Avatar       string         `gorm:"column:avatar" json:"avatar"`
    Power        int            `gorm:"column:power" json:"power"`
    AccountType  string         `gorm:"column:accountType;default:base" json:"accountType"`
    StuNum       string         `gorm:"column:stuNum" json:"stuNum"`
    StuName      string         `gorm:"column:stuName" json:"stuName"`
    StuPwd       string         `gorm:"column:stuPwd" json:"-"`
    StuCla       string         `gorm:"column:stuCla" json:"stuCla"`
    StuIsCheck   bool           `gorm:"column:stuIsCheck" json:"stuIsCheck"`
    School       string         `gorm:"column:school" json:"school"`
    Tag          string         `gorm:"column:tag;default:student" json:"tag"`
    Gender       string         `gorm:"column:gender;default:保密" json:"gender"`
    RootUserID   int64          `gorm:"column:rootUserId" json:"rootUserId"`
    LastSwitchID *int64         `gorm:"column:lastSwitchId" json:"lastSwitchId"`
    Signature    string         `gorm:"column:signature" json:"signature"`
    CreatedAt    time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
    CreatedBy    int64          `gorm:"column:createdBy" json:"createdBy"`
    UpdatedAt    time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
    UpdatedBy    int64          `gorm:"column:updatedBy" json:"updatedBy"`
    DeletedAt    gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
    DeletedBy    int64          `gorm:"column:deletedBy" json:"-"`
}

func (User) TableName() string { return "campus_user" }
```

| 枚举字段 | 值 | 含义 |
|---------|---|------|
| AccountType | `"base"` | 基座账号（微信用户） |
| AccountType | `"anonymous"` | 匿名身份 |
| AccountType | `"official"` | 官方号 |
| Tag | `"student"` | 学生（默认） |
| Tag | `"organization"` | 组织 |
| Tag | `"school-official"` | 学校官方 |
| Tag | `"blogger"` | 博主 |
| Tag | `"faculty"` | 教职工 |
| Tag | `"merchant"` | 商户 |
| Gender | `"男"` / `"女"` / `"保密"` | 性别 |
| Power | 0 | 普通用户 |
| Power | 2 | 普通管理员（0b10） |
| Power | 999 | 超级管理员 |

### 1.2 admin — `internal/user/model.go`

```go
type Admin struct {
    ID       int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    UserID   int64  `gorm:"column:userId;uniqueIndex" json:"userId"`
    Username string `gorm:"column:username;uniqueIndex" json:"username"`
    Password string `gorm:"column:password" json:"-"` // char(32) MD5
    Power    int    `gorm:"column:power;default:2" json:"power"`
}

func (Admin) TableName() string { return "admin" }
```

**Admin.Power**：2=普通管理员，999=超级管理员。与 User.Power 含义一致。

### 1.3 conversations — `internal/chat/model.go`

```go
type Conversation struct {
    ID                  int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Type                int       `gorm:"column:type" json:"type"`
    LastMessageContent  string    `gorm:"column:lastMessageContent" json:"lastMessageContent"`
    LastMessageSenderID int64     `gorm:"column:lastMessageSenderId" json:"lastMessageSenderId"`
    LastMessageSentAt   time.Time `gorm:"column:lastMessageSentAt" json:"lastMessageSentAt"`
    CreatedAt           time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
    UpdatedAt           time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (Conversation) TableName() string { return "conversations" }
```

Type: 1=单聊, 2=群聊

### 1.4 conversation_members — `internal/chat/model.go`

```go
type ConversationMember struct {
    ID                int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    ConversationID    int64 `gorm:"column:conversationId" json:"conversationId"`
    UserID            int64 `gorm:"column:userId" json:"userId"`
    LastReadMessageID int64 `gorm:"column:lastReadMessageId" json:"lastReadMessageId"`
    UnreadCount       int   `gorm:"column:unreadCount" json:"unreadCount"`
}

func (ConversationMember) TableName() string { return "conversation_members" }
```

### 1.5 campus_ad — `internal/other/model.go`

```go
type Ad struct {
    ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    BannerURL string         `gorm:"column:bannerUrl" json:"bannerUrl"`
    TopicID   string         `gorm:"column:topicId" json:"topicId"`
    Level     int            `gorm:"column:level" json:"level"`
    IsOk      bool           `gorm:"column:isOk" json:"isOk"`
    CreatedAt time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
    UpdatedAt time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
    DeletedAt gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (Ad) TableName() string { return "campus_ad" }
```

### 1.6 campus_notice — `internal/other/model.go`

```go
type Notice struct {
    ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Content   string         `gorm:"column:content" json:"content"`
    CreatedBy int64          `gorm:"column:createdBy" json:"createdBy"`
    UpdatedBy int64          `gorm:"column:updatedBy" json:"updatedBy"`
    DeletedBy int64          `gorm:"column:deletedBy" json:"deletedBy"`
    CreatedAt time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
    UpdatedAt time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
    DeletedAt gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (Notice) TableName() string { return "campus_notice" }
```

### 1.7 sensitive_words — `internal/other/model.go`

```go
type SensitiveWord struct {
    ID   int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Word string `gorm:"column:word" json:"word"`
}

func (SensitiveWord) TableName() string { return "sensitive_words" }
```

### 1.8 campus_vote_info — `internal/other/model.go`

```go
type VoteInfo struct {
    ID            int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Title         string         `gorm:"column:title" json:"title"`
    Content       string         `gorm:"column:content" json:"content"`
    AccessDraft   bool           `gorm:"column:accessDraft" json:"accessDraft"`
    AccessEndTime time.Time      `gorm:"column:accessEndTime" json:"accessEndTime"`
    VoteStart     time.Time      `gorm:"column:voteStart" json:"voteStart"`
    VoteEnd       time.Time      `gorm:"column:voteEnd" json:"voteEnd"`
    Mode          int            `gorm:"column:mode" json:"mode"`
    OptionType    int            `gorm:"column:optionType" json:"optionType"`
    UserID        int64          `gorm:"column:userId" json:"userId"`
    CreatedAt     time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
    UpdatedAt     time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
    DeletedAt     gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (VoteInfo) TableName() string { return "campus_vote_info" }
```

Mode: 1=文字, 2=图片。OptionType: 1=单选, 2=多选。

### 1.9 campus_vote_option — `internal/other/model.go`

```go
type VoteOption struct {
    ID         int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    VoteInfoID int64          `gorm:"column:voteInfoId" json:"voteInfoId"`
    VoteText   string         `gorm:"column:voteText" json:"voteText"`
    VoteImg    string         `gorm:"column:voteImg" json:"voteImg"`
    IsOk       bool           `gorm:"column:isOk" json:"isOk"`
    CreatedAt  time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
    UpdatedAt  time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
    DeletedAt  gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (VoteOption) TableName() string { return "campus_vote_option" }
```

### 1.10 campus_vote_ans — `internal/other/model.go`

```go
type VoteAns struct {
    ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    VoteInfoID   int64     `gorm:"column:voteInfoId" json:"voteInfoId"`
    VoteDate     time.Time `gorm:"column:voteDate" json:"voteDate"`
    VoteUserID   int64     `gorm:"column:voteUserId" json:"voteUserId"`
    VoteOptionID int64     `gorm:"column:voteOptionId" json:"voteOptionId"`
}

func (VoteAns) TableName() string { return "campus_vote_ans" }
```

### 1.11 exp_detail — `internal/level/model.go`

```go
type ExpDetail struct {
    ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    UserID     int64     `gorm:"column:user_id" json:"userId"`
    GetExpDate time.Time `gorm:"column:get_exp_date" json:"getExpDate"`
    GetExp     int       `gorm:"column:get_exp" json:"getExp"`
}

func (ExpDetail) TableName() string { return "exp_detail" }
```

### 1.12 event_data — `internal/event/model.go`

```go
type Event struct {
    EventID      int64     `gorm:"column:eventId;primaryKey;autoIncrement" json:"eventId"`
    EventType    string    `gorm:"column:eventType" json:"eventType"`
    EventInfo    string    `gorm:"column:eventInfo" json:"eventInfo"`
    EventContent string    `gorm:"column:eventContent" json:"eventContent"`
    UserID       int64     `gorm:"column:userId" json:"userId"`
    TriggerTime  time.Time `gorm:"column:triggerTime" json:"triggerTime"`
}

func (Event) TableName() string { return "event_data" }
```

### 1.13 campus_user_course — `internal/school/model.go`

```go
type UserCourse struct {
    ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    UserID    int64     `gorm:"column:userId" json:"userId"`
    Status    int       `gorm:"column:status" json:"status"`
    Term      string    `gorm:"column:term" json:"term"`
    Week      int       `gorm:"column:week" json:"week"`
    Course    string    `gorm:"column:course;type:text" json:"course"`
    UpdatedAt time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (UserCourse) TableName() string { return "campus_user_course" }
```

Status: 1=抓取中, 2=完成, 3=失败。Course: JSON 字符串。

### 1.14 campus_task — `internal/other/model.go`

```go
type Task struct {
    ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Status    int            `gorm:"column:status" json:"status"`
    Detail    string         `gorm:"column:detail" json:"detail"`
    Parent    int64          `gorm:"column:parent" json:"parent"`
    Func      string         `gorm:"column:func" json:"func"`
    CreatedAt time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
    UpdatedAt time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
    DeletedAt gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (Task) TableName() string { return "campus_task" }
```

Status: 0=初始化, 1=运行中, 2=成功, 3=失败。

---

## 2. MongoDB 文档（18 个集合）

### 2.1 campus_topic — `internal/topic/model.go`

```go
type Topic struct {
    ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    ThemeID       string             `bson:"themeId" json:"themeId"`
    UserID        string             `bson:"userId" json:"userId"`
    Title         string             `bson:"title" json:"title"`
    Content       string             `bson:"content" json:"content"`
    Imgs          []string           `bson:"imgs" json:"imgs"`
    HasCheck      bool               `bson:"hasCheck" json:"hasCheck"`
    VisitedNum    int64              `bson:"visitedNum" json:"visitedNum"`
    LikeNum       int64              `bson:"likeNum" json:"likeNum"`
    CommentNum    int64              `bson:"commentNum" json:"commentNum"`
    CollectionNum int64              `bson:"collectionNum" json:"collectionNum"`
    Ext           interface{}        `bson:"ext,omitempty" json:"ext"`
    AccountType   int                `bson:"accountType" json:"accountType"`
    NickName      string             `bson:"nickName" json:"nickName"`
    Avatar        string             `bson:"avatar" json:"avatar"`
    // 瞬态字段（不存 DB，查询后填充）
    HasLike       bool               `bson:"-" json:"hasLike"`
    HasCollection bool               `bson:"-" json:"hasCollection"`
}
```

索引：`themeId`、`userId`、`hasCheck` 各建单字段索引。

软删除：`hasCheck = false` 表示已删/未审核。查询时 `hasCheck: true` 过滤。

**注意**：`Imgs` 字段序列化时必须确保 `[]string{}` 而非 `nil`（否则 JSON 输出 `null` 而非 `[]`）。

### 2.2 campus_comment — `internal/comment/model.go`

```go
type Comment struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    TopicID     string             `bson:"topicId" json:"topicId"`
    Comment     string             `bson:"comment" json:"comment"`
    CreatedTime time.Time          `bson:"createdTime" json:"createdTime"`
    User        CommentUser        `bson:"user" json:"user"`
    Parent      *CommentUser       `bson:"parent,omitempty" json:"parent"`
    ParentCmtID string             `bson:"parentCmtId,omitempty" json:"parentCmtId"`
    RootCmtID   string             `bson:"rootCmtId,omitempty" json:"rootCmtId"`
    IsAuthor    bool               `bson:"isAuthor" json:"isAuthor"`
    LikeNum     int64              `bson:"likeNum" json:"likeNum"`
    CommentNum  int64              `bson:"commentNum" json:"commentNum"`
    HasCheck    bool               `bson:"hasCheck" json:"hasCheck"`
    // 瞬态
    HasLike     bool               `bson:"-" json:"hasLike"`
}

type CommentUser struct {
    UserID      string `bson:"userId" json:"userId"`
    NickName    string `bson:"nickName" json:"nickName"`
    Avatar      string `bson:"avatar" json:"avatar"`
    AccountType int    `bson:"accountType" json:"accountType"`
}
```

索引：`topicId` 单字段索引。

### 2.3 campus_theme — `internal/theme/model.go`

```go
type Theme struct {
    ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Name              string             `bson:"name" json:"name"`
    CategoryName      string             `bson:"category_name" json:"categoryName"`
    NeedSearch        bool               `bson:"needSearch" json:"needSearch"`
    NeedSuggest       bool               `bson:"needSuggest" json:"needSuggest"`
    SuggestBasicScore int64              `bson:"suggestBasicScore" json:"suggestBasicScore"`
    SuggestNumber     int                `bson:"suggestNumber" json:"suggestNumber"`
    SuggestSetName    string             `bson:"suggestSetName" json:"suggestSetName"`
    SuggestType       string             `bson:"suggestType" json:"suggestType"`
}
```

索引：`name` 单字段索引。

### 2.4 campus_topic_search — `internal/topic/model.go`

```go
type TopicSearch struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    TopicID   string             `bson:"topicId" json:"topicId"`
    ThemeName string             `bson:"themeName" json:"themeName"`
    Title     string             `bson:"title" json:"title"`
    Content   string             `bson:"content" json:"content"`
}
```

索引：
- `topicId` 单字段
- `themeName` 单字段
- `title` + `content` + `themeName` text 复合索引

**注意**：title 和 content 存储的是**分词后**的字符串（gse 分词），非原文。

### 2.5 campus_topic_like — `internal/topic/model.go`

```go
type TopicLike struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID      string             `bson:"userId" json:"userId"`
    ThemeName   string             `bson:"themeName" json:"themeName"`
    AccountType int                `bson:"accountType" json:"accountType"`
    TopicIDs    []string           `bson:"topicIds" json:"topicIds"`
}
```

索引：`userId` + `themeName` 复合唯一索引。

设计：每个用户每个主题一条记录，`topicIds` 数组存已点赞帖子 ID。点赞 = `$push`，取消 = `$pull`。

### 2.6 campus_topic_collection — `internal/topic/model.go`

```go
type TopicCollection struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID      string             `bson:"userId" json:"userId"`
    ThemeName   string             `bson:"themeName" json:"themeName"`
    AccountType int                `bson:"accountType" json:"accountType"`
    TopicIDs    []string           `bson:"topicIds" json:"topicIds"`
}
```

结构与 TopicLike 相同，索引相同。

### 2.7 campus_comment_like — `internal/comment/model.go`

```go
type CommentLike struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    CommentID string             `bson:"commentId" json:"commentId"`
    UserIDs   []string           `bson:"userIds" json:"userIds"`
}
```

索引：`commentId` 单字段。每条评论一个文档，`userIds` 记录点赞用户。

### 2.8 campus_follow — `internal/user/model.go`

```go
type Follow struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    FollowerID  string             `bson:"followerId" json:"followerId"`
    FollowingID string             `bson:"followingId" json:"followingId"`
    FollowAt    time.Time          `bson:"followAt" json:"followAt"`
}
```

索引：`followerId` + `followingId` 复合唯一索引。

### 2.9 campus_user_blacklist — `internal/user/model.go`

```go
type UserBlacklist struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    BlockedUserIDs []string           `bson:"blocked_user_ids" json:"blockedUserIds"`
    CreatedTime    time.Time          `bson:"created_time" json:"createdTime"`
    UpdatedTime    time.Time          `bson:"updated_time" json:"updatedTime"`
}
```

### 2.10 campus_messages — `internal/chat/model.go`

```go
type Message struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    MessageID      int64              `bson:"message_id" json:"messageId"`
    ConversationID int64              `bson:"conversation_id" json:"conversationId"`
    ReceiverID     int64              `bson:"receiver_id" json:"receiverId"`
    SenderID       int64              `bson:"sender_id" json:"senderId"`
    Content        string             `bson:"content" json:"content"`
    SentAt         time.Time          `bson:"sentAt" json:"sentAt"`
}
```

### 2.11 campus_notifications — `internal/chat/model.go`

```go
type Notification struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID    string             `bson:"userId" json:"userId"`
    Type      string             `bson:"type" json:"type"`
    Content   interface{}        `bson:"content" json:"content"`
    CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
    IsRead    bool               `bson:"isRead" json:"isRead"`
}
```

Type: `"like"` / `"comment"` / `"follow"` / `"system"`

### 2.12 campus_file — `internal/file/model.go`

```go
type File struct {
    ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    MD5      string             `bson:"md5" json:"md5"`
    IsPublic bool               `bson:"isPublic" json:"isPublic"`
    UserID   string             `bson:"userId" json:"userId"`
    RefCount int                `bson:"refCount" json:"refCount"`
}
```

索引：`md5` 单字段唯一索引、`userId` 单字段索引。

### 2.13 campus_term — `internal/school/model.go`

```go
type Term struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    TermName  string             `bson:"termName" json:"termName"`
    StartDate time.Time          `bson:"startDate" json:"startDate"`
    EndDate   time.Time          `bson:"endDate" json:"endDate"`
}
```

### 2.14 campus_cur_term — `internal/school/model.go`

```go
type CurTerm struct {
    ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    TermID string             `bson:"termId" json:"termId"`
}
```

### 2.15 campus_official_certification — `internal/user/model.go`

```go
type OfficialCertification struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID    string             `bson:"userId" json:"userId"`
    Name      string             `bson:"name" json:"name"`
    Reason    string             `bson:"reason" json:"reason"`
    Status    int                `bson:"status" json:"status"`
    CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
```

Status: 0=待审核, 1=通过, 2=拒绝

### 2.16 campus_merchant_theme — `internal/other/model.go`

```go
type MerchantTheme struct {
    ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    ThemeID string             `bson:"themeId" json:"themeId"`
}
```

### 2.17 campus_frontend_support — `internal/other/model.go`

```go
type FrontendSupport struct {
    ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Key   string             `bson:"key" json:"key"`
    Value interface{}        `bson:"value" json:"value"`
}
```

### 2.18 其他 MongoDB 集合 — 各自域包

```go
// internal/other/model.go — 举报评论
type ReportComment struct {
    ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    CommentID  string             `bson:"commentId" json:"commentId"`
    TopicID    string             `bson:"topicId" json:"topicId"`
    ReporterID string             `bson:"reporterId" json:"reporterId"`
    Reason     string             `bson:"reason" json:"reason"`
    Status     int                `bson:"status" json:"status"`
    CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
}

// internal/school/model.go — 课程缓存
type Course struct {
    ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Key      string             `bson:"key" json:"key"`
    FilePath string             `bson:"filePath" json:"filePath"`
    Val      []byte             `bson:"val" json:"val"`
}

// internal/file/model.go — 压缩文件记录
type CompressFile struct {
    ID  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    MD5 string             `bson:"md5" json:"md5"`
    URL string             `bson:"url" json:"url"`
}

// internal/mq/model.go — MQ 失败日志
type MQLog struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    CreatedTime time.Time          `bson:"createdTime" json:"createdTime"`
    Type        string             `bson:"type" json:"type"`
    Data        interface{}        `bson:"data" json:"data"`
}
// Type: "to_broker_fail" 或 "to_queue_fail"
```

---

## 3. 模型归属汇总

| 包 | model.go 中包含的 struct | MySQL 表 | MongoDB 集合 |
|----|------------------------|---------|-------------|
| `user/` | User, Admin, Follow, UserBlacklist, OfficialCertification | campus_user, admin | campus_follow, campus_user_blacklist, campus_official_certification |
| `topic/` | Topic, TopicSearch, TopicLike, TopicCollection | - | campus_topic, campus_topic_search, campus_topic_like, campus_topic_collection |
| `comment/` | Comment, CommentUser, CommentLike | - | campus_comment, campus_comment_like |
| `theme/` | Theme | - | campus_theme |
| `file/` | File, CompressFile | - | campus_file, campus_compress_file |
| `chat/` | Conversation, ConversationMember, Message, Notification | conversations, conversation_members | campus_messages, campus_notifications |
| `level/` | ExpDetail | exp_detail | - |
| `school/` | UserCourse, Term, CurTerm, Course | campus_user_course | campus_term, campus_cur_term, campus_course |
| `event/` | Event | event_data | - |
| `other/` | Ad, Notice, SensitiveWord, VoteInfo, VoteOption, VoteAns, Task, ReportComment, MerchantTheme, FrontendSupport | campus_ad, campus_notice, sensitive_words, campus_vote_info, campus_vote_option, campus_vote_ans, campus_task | campus_report_comment, campus_merchant_theme, campus_frontend_support |
| `mq/` | MQLog | - | campus_mq |

---

## 4. Redis Key 定义 — `internal/pkg/rediskey/keys.go`

```go
package rediskey

import "fmt"

// ========== Token ==========
const (
    TokenPrefix        = "campus:token:"
    RefreshTokenPrefix = "campus:refresh_token:"
)

func Token(sha1 string) string       { return TokenPrefix + sha1 }
func RefreshToken(sha1 string) string { return RefreshTokenPrefix + sha1 }

const (
    TokenStatusOK   = "OK"
    TokenStatusUsed = "USED"
)

// ========== 用户 ==========
const (
    UserPrefix     = "campus:user:"
    UserExpPrefix  = "campus:userExp:"
    UserSignPrefix = "campus:userSign:"
    ExpDetailKey   = "campus:expDetail:DETAIL_KEY"
    GlobalBlacklist = "campus:global_blacklist"
)

func User(id int64) string             { return fmt.Sprintf("%s%d", UserPrefix, id) }
func UserExp(id int64) string          { return fmt.Sprintf("%s%d", UserExpPrefix, id) }
func UserSign(yearMonth string) string { return UserSignPrefix + yearMonth }

// ========== 活跃用户 (HyperLogLog) ==========
const ActiveDayPrefix = "campus:active:day:"

func ActiveDay(date string) string { return ActiveDayPrefix + date }

// ========== MQ 消息去重 ==========
const (
    AddMsgCache       = "campus:AMC:"
    DeleteMsgCache    = "campus:DMC:"
    UpdateMsgCache    = "campus:UMC:"
    TopicCreateCache  = "campus:TCC:"
    TopicInfoCache    = "campus:TIC:"
    DeleteTopicCache  = "campus:DTC:"
    UpdateTopicSearch = "campus:UTS:"
    AddTopicSearch    = "campus:ATS:"
    GetAllCourse      = "campus:GAC:"
    NotifyCache       = "campus:NOTIFY:"
)

// ========== 推荐排行 ==========
const (
    SuggestRankPrefix = "rank:"
    SuggestCurKey     = "suggest:cur"
    SuggestPrevKey    = "suggest:prev"
    SuggestCountKey   = "suggest:cnt"
)

func SuggestRank(setName string) string { return SuggestRankPrefix + setName }

// ========== 缓存 ==========
const (
    SuggestTopicListKey  = "campus:suggest_topic_list"
    UserCoursePrefix     = "campus:userCourse:"
    ControllerTimePrefix = "campus:controllerTime:"
    EventKey             = "campus:event:EVENT_KEY"
)

func UserCourse(userId int64, term string, week int) string {
    return fmt.Sprintf("%s%d:%s:%d", UserCoursePrefix, userId, term, week)
}

// ========== 管理员登录 ==========
const (
    AdminLoginLockPrefix    = "admin:login:lock:"
    AdminLoginFailPrefix    = "admin:login:fail:count:"
)

func AdminLoginLock(username string) string    { return AdminLoginLockPrefix + username }
func AdminLoginFail(username string) string    { return AdminLoginFailPrefix + username }

// ========== MQ UUID ==========
const MQUUIDKey = "uuid"
```

### Redis 数据类型与 TTL

| Key Pattern | 类型 | TTL | 说明 |
|-------------|------|-----|------|
| `campus:token:{sha1}` | String | jwt.token_minutes | 访问令牌 |
| `campus:refresh_token:{sha1}` | String | jwt.refresh_token_minutes | 刷新令牌（值为 OK 或 USED） |
| `campus:userExp:{id}` | String(JSON) | 30min（有值）/ 5s（空值） | 用户经验缓存 |
| `campus:userSign:{yyyyMM}` | Bitmap | 不过期 | 签到位图，offset = userID * 31 + day |
| `campus:expDetail:DETAIL_KEY` | List | 不过期 | 经验明细写入队列 |
| `campus:event:EVENT_KEY` | List | 不过期 | 埋点缓冲队列 |
| `campus:global_blacklist` | Set | 不过期 | 全局黑名单 userID 集合 |
| `campus:active:day:{yyyyMMdd}` | HyperLogLog | 40 天 | 每日活跃用户 |
| `campus:AMC:{id}` 等 MQ 去重 | String(int) | 3 天 | 消息处理状态 0/1/2 |
| `rank:{setName}` | Sorted Set | 不过期（定时清理） | 推荐排行分数 |
| `suggest:cur` / `suggest:prev` | String | 不过期 | 当前/上一推荐排行 key |
| `campus:suggest_topic_list` | Hash | 不过期 | 推荐列表分页缓存 |
| `campus:userCourse:{id}:{term}:{week}` | String(JSON) | 不过期 | 课表缓存 |
| `uuid` | String(int) | 不过期 | MQ UUID 自增计数器 |
| `admin:login:lock:{username}` | String | 24h | 管理员登录锁定标记（10 次失败后触发） |
| `admin:login:fail:count:{username}` | String(int) | 24h | 管理员登录失败计数器 |

---

## 5. RabbitMQ 队列定义 — `internal/mq/config.go`

```go
const (
    Exchange    = "campus.exchange"
    DieExchange = "campus.die_exchange"
)

const (
    QueueCommentUpdate     = "campus.comment_update"
    QueueTopicUpdate       = "campus.topic_update"
    QueueTopicDelete       = "campus.topic_delete"
    QueueCommentDelete     = "campus.comment_delete"
    QueueCommentAdd        = "campus.comment_add"
    QueueTopicSearchAdd    = "campus.topic_search_add"
    QueueTopicSearchUpdate = "campus.topic_search_update"
    QueueTopicSearchDel    = "campus.topic_search_del"
    QueueTopicCheck        = "campus.topic_check"
    QueueGetCourse         = "campus.get_course"
    QueueNotifyUser        = "campus.notify_user"
    QueueDie               = "campus.die"
)

const (
    KeyUpdateCommentUser = "update_cmt_user"
    KeyUpdateTopicUser   = "update_topic_user"
    KeyDeleteTopic       = "delete_topic"
    KeyDeleteComment     = "delete_comment"
    KeyAddComment        = "add_comment"
    KeyAddTopicSearch    = "add_topic_search"
    KeyUpdateTopicSearch = "update_topic_search"
    KeyDelTopicSearch    = "del_topic_search"
    KeyTopicCheck        = "topic_check"
    KeyGetCourse         = "get_course"
    KeyNotifyUser        = "notify_user"
    KeyDie               = "die"
)

const (
    MsgPrev = 0  // 预处理
    MsgIng  = 1  // 处理中
    MsgPost = 2  // 已处理
)
```

Exchange 类型：全部 Direct Exchange。

### 队列消费者归属（全部由 ecampus 消费）

| 队列 | Consumer 位置 | 功能 |
|------|-------------|------|
| campus.topic_check | `mq/consumer.go` | 帖子内容审核（调微信安全 API） |
| campus.comment_add | `mq/consumer.go` | 评论创建 + 内容审核 + 通知 |
| campus.topic_search_add | `mq/consumer.go` | 添加搜索索引（分词） |
| campus.topic_search_update | `mq/consumer.go` | 更新搜索索引 |
| campus.topic_search_del | `mq/consumer.go` | 删除搜索索引 |
| campus.topic_update | `mq/consumer.go` | 更新帖子中的用户信息 |
| campus.topic_delete | `mq/consumer.go` | 级联删除帖子相关数据 |
| campus.comment_update | `mq/consumer.go` | 更新评论中的用户信息 |
| campus.comment_delete | `mq/consumer.go` | 级联删除评论相关数据 |
| campus.get_course | `mq/consumer.go` | 异步抓取课程表 |
| campus.notify_user | `mq/consumer.go` | 发送通知 + WebSocket 推送 |
| campus.die | `mq/consumer.go` | 死信队列处理 |

---

## 6. 分页类型 — `internal/pkg/result/page.go`

```go
// MySQL 分页（对应 MyBatis-Plus Page，JSON key 必须一致）
type PageResult[T any] struct {
    Records []T   `json:"records"`
    Total   int64 `json:"total"`
    Current int   `json:"current"`
    Size    int   `json:"size"`
    Pages   int   `json:"pages"`
}

func NewPage[T any](records []T, total int64, current, size int) *PageResult[T] {
    pages := int(total) / size
    if int(total)%size > 0 { pages++ }
    return &PageResult[T]{
        Records: EnsureSlice(records),
        Total: total, Current: current, Size: size, Pages: pages,
    }
}

// 自定义分页（MongoDB 等）
type CusPage[T any] struct {
    Data    []T   `json:"data"`
    Current int   `json:"current"`
    Total   int64 `json:"total"`
    Size    int   `json:"size"`
}

func NewCusPage[T any](data []T, total int64, current, size int) *CusPage[T] {
    return &CusPage[T]{
        Data: EnsureSlice(data),
        Total: total, Current: current, Size: size,
    }
}
```

**关键**：`records`/`data` 字段必须用 `EnsureSlice()` 确保空数组 `[]` 而非 `null`，与 Java FastJSON 行为一致。

---

## 7. 请求/响应专用结构体

> 只在请求 shape 确实不同于 model 时才定义。定义在各域的 `model.go` 或 `model_req.go` 中。

### `internal/user/model.go`

```go
type AdminLoginReq struct {
    Username          string `json:"username" binding:"required"`
    Password          string `json:"password" binding:"required"`
    SecondaryPassword string `json:"secondaryPassword" binding:"required"`
}

type UserProfile struct {
    User
    FollowerCount  int64 `json:"followerCount"`
    FollowingCount int64 `json:"followingCount"`
    LikeCount      int64 `json:"likeCount"`
    TopicCount     int64 `json:"topicCount"`
    IsFollowing    bool  `json:"isFollowing"`
}
```

### `internal/topic/model.go`

```go
type CreateTopicReq struct {
    ThemeID  string      `json:"themeId" binding:"required"`
    Title    string      `json:"title" binding:"required"`
    Content  string      `json:"content" binding:"required"`
    Imgs     []string    `json:"imgs"`
    Ext      interface{} `json:"ext"`
    NickName string      `json:"nickName"`
    Avatar   string      `json:"avatar"`
}

type SuggestListVO struct {
    Total   int64   `json:"total"`
    CurPage int     `json:"curPage"`
    Size    int     `json:"size"`
    Data    []Topic `json:"data"`
}
```

### `internal/level/model.go`

```go
type SignDetailVO struct {
    Exp         int  `json:"exp"`
    SignDays    int  `json:"signDays"`
    TodaySigned bool `json:"todaySigned"`
}
```

### `internal/monitor/model.go`

```go
type CacheStats struct {
    CacheName string `json:"cacheName"`
    Size      int64  `json:"size"`
    HitCount  int64  `json:"hitCount"`
    MissCount int64  `json:"missCount"`
    HitRate   float64 `json:"hitRate"`
}
```

### `internal/mq/model.go`（MQ 消息结构）

```go
type MQMessage struct {
    UniqueID int64       `json:"uniqueId"`
    Data     interface{} `json:"data"`
}

type TopicCheckMsg struct {
    TopicID string `json:"topicId"`
}

type AddCommentMsg struct {
    Comment comment.Comment `json:"comment"`
}

type AddTopicSearchMsg struct {
    TopicID string `json:"topicId"`
}

type CourseMsg struct {
    UserID int64  `json:"userId"`
    StuNum string `json:"stuNum"`
    StuPwd string `json:"stuPwd"`
    Term   string `json:"term"`
    Week   int    `json:"week"`
}

type NotifyMsg struct {
    TargetUserID string      `json:"targetUserId"`
    Type         string      `json:"type"`
    Content      interface{} `json:"content"`
}
```
