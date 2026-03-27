# Batch 8 审计报告：路由总检 + 全局交叉验证

审计日期：2026-03-26

审计前提：
- 本次重构切换后，不存在 Java 与 Go 双版本同时部署。
- 旧 Redis 缓存允许在切换前清空，但 MySQL / MongoDB 现网数据需要被 Go 直接读取。
- 主报告仅计入 Go 独立部署场景下必然触发的问题；本轮未把仅依赖运行时外部数据、仓库内无法还原的情况混入必须修复项。

补充说明：
- 路由总检以 Java Controller 注册出来的活跃 REST 端点为主，同时额外纳入 Java `WebSocketConfig` 注册的 `/chat`，因为它是现网对外暴露的活跃入口。
- 路径参数占位符的语法差异（Java `{id}` vs Go `:id`）不计为问题；只有会导致客户端命中不同路径或收到重定向的差异才计入。
- `aop_permission` 表只见结构定义，仓库中没有初始化数据，因此“哪些接口应加经验值”无法从仓库内独立复原；这一点在文末单独标注为“未验证，需人工确认”。

## 模块：路由总检

### 活跃 API 端点清单

| # | 端点 | 方法 | Java Controller | Go 服务 | Go Handler | 状态 |
|---|------|------|----------------|---------|------------|------|
| 1 | `/api/conversation` | GET | `ConversationController (ConversationController.java:23)` | `ecampus` | `handler.go:33` | ✅ |
| 2 | `/api/conversation/conversation_enter` | PUT | `ConversationController (ConversationController.java:34)` | `ecampus` | `handler.go:42` | ✅ |
| 3 | `/api/conversation/{conversation_id}/unread_count` | GET | `ConversationController (ConversationController.java:42)` | `ecampus` | `handler.go:54` | ✅ |
| 4 | `/api/conversation/conversation_query` | GET | `ConversationController (ConversationController.java:48)` | `ecampus` | `handler.go:63` | ✅ |
| 5 | `/api/conversation/profile_by_conversation_id` | GET | `ConversationController (ConversationController.java:54)` | `ecampus` | `handler.go:72` | ✅ |
| 6 | `/api/conversation/{conversation_id}` | DELETE | `ConversationController (ConversationController.java:59)` | `ecampus` | `handler.go:107` | ✅ |
| 7 | `/api/message/{last_message_id}` | GET | `MessageController (MessageController.java:25)` | `ecampus` | `handler.go:115` | ✅ |
| 8 | `/api/message/history_messages` | GET | `MessageController (MessageController.java:31)` | `ecampus` | `handler.go:129` | ✅ |
| 9 | `/api/message/unread_messages` | GET | `MessageController (MessageController.java:48)` | `ecampus` | `handler.go:156` | ✅ |
| 10 | `/api/notify` | GET | `NotifyController (NotifyController.java:24)` | `ecampus` | `handler.go:165` | ✅ |
| 11 | `/api/notify/{type}/haveUnread` | GET | `NotifyController (NotifyController.java:41)` | `ecampus` | `handler.go:179` | ✅ |
| 12 | `/api/notify/{type}` | GET | `NotifyController (NotifyController.java:47)` | `ecampus` | `handler.go:188` | ✅ |
| 13 | `/file/upload` | POST | `FileController (FileController.java:72)` | `ecampus` | `handler.go:40` | ✅ |
| 14 | `/file/{md5}` | GET | `FileController (FileController.java:124)` | `ecampus` | `handler.go:21` | ✅ |
| 15 | `/file/del/{md5}` | DELETE | `FileController (FileController.java:159)` | `ecampus` | `handler.go:55` | ✅ |
| 16 | `/file` | GET | `FileController (FileController.java:181)` | `ecampus` | `handler.go:30` | ✅ |
| 17 | `/admin/file` | POST | `AdmFileController (AdmFileController.java:40)` | `ecampus-crm` | `admin.go:15` | ✅ |
| 18 | `/admin/file` | GET | `AdmFileController (AdmFileController.java:64)` | `ecampus-crm` | `admin.go:27` | ✅ |
| 19 | `/api/event/` | POST | `EventController (EventController.java:35)` | `ecampus` | `handler.go:15` | ⚠️ 路径不一致 |
| 20 | `/admin/event/{id}` | DELETE | `AdmEventDataController (AdmEventDataController.java:41)` | `ecampus-crm` | `admin.go:19` | ✅ |
| 21 | `/admin/event/{id}` | PUT | `AdmEventDataController (AdmEventDataController.java:52)` | `ecampus-crm` | `admin.go:36` | ✅ |
| 22 | `/admin/event/{id}` | GET | `AdmEventDataController (AdmEventDataController.java:66)` | `ecampus-crm` | `admin.go:57` | ✅ |
| 23 | `/admin/event/list` | GET | `AdmEventDataController (AdmEventDataController.java:75)` | `ecampus-crm` | `admin.go:75` | ✅ |
| 24 | `/api/getUserSignDetail` | GET | `LevelController (LevelController.java:35)` | `ecampus` | `handler.go:21` | ✅ |
| 25 | `/api/testAop` | GET | `LevelController (LevelController.java:47)` | `—` | `—` | ❌ 缺失 |
| 26 | `/api/sign_in` | POST | `LevelController (LevelController.java:55)` | `ecampus` | `handler.go:30` | ✅ |
| 27 | `/api/exp+3/{id}` | POST | `LevelController (LevelController.java:68)` | `—` | `—` | ❌ 缺失 |
| 28 | `/api/UserExp` | GET | `LevelController (LevelController.java:74)` | `ecampus` | `handler.go:38` | ✅ |
| 29 | `/admin/local_cache/all_key` | GET | `LocalCacheController (LocalCacheController.java:31)` | `ecampus-crm` | `admin.go:15` | ✅ |
| 30 | `/admin/local_cache/stats` | GET | `LocalCacheController (LocalCacheController.java:38)` | `ecampus-crm` | `admin.go:24` | ✅ |
| 31 | `/admin/task` | POST | `TaskController (TaskController.java:27)` | `ecampus-crm` | `task_admin.go:11` | ✅ |
| 32 | `/admin/task/{id}` | DELETE | `TaskController (TaskController.java:33)` | `ecampus-crm` | `task_admin.go:23` | ✅ |
| 33 | `/admin/task/{id}` | PUT | `TaskController (TaskController.java:41)` | `ecampus-crm` | `task_admin.go:40` | ✅ |
| 34 | `/admin/task/{id}` | GET | `TaskController (TaskController.java:52)` | `ecampus-crm` | `task_admin.go:61` | ✅ |
| 35 | `/admin/task/list` | GET | `TaskController (TaskController.java:61)` | `ecampus-crm` | `task_admin.go:79` | ✅ |
| 36 | `/api/ad/list_level` | GET | `AdController (AdController.java:23)` | `ecampus` | `ad.go:11` | ✅ |
| 37 | `/api/notice/list` | GET | `NoticeController (NoticeController.java:35)` | `ecampus` | `notice.go:9` | ✅ |
| 38 | `/api/vote/list` | GET | `VoteController (VoteController.java:52)` | `ecampus` | `vote.go:12` | ✅ |
| 39 | `/api/vote/draft/{info_id}` | GET | `VoteController (VoteController.java:61)` | `ecampus` | `vote.go:22` | ✅ |
| 40 | `/api/vote/draft/{info_id}` | PUT | `VoteController (VoteController.java:79)` | `ecampus` | `vote.go:36` | ✅ |
| 41 | `/api/vote` | POST | `VoteController (VoteController.java:102)` | `ecampus` | `vote.go:53` | ✅ |
| 42 | `/api/vote/{info_id}` | POST | `VoteController (VoteController.java:133)` | `ecampus` | `vote.go:66` | ✅ |
| 43 | `/api/vote/vote/{info_id}` | POST | `VoteController (VoteController.java:151)` | `ecampus` | `vote.go:84` | ✅ |
| 44 | `/admin/ad` | POST | `AdmAdController (AdmAdController.java:34)` | `ecampus-crm` | `ad_admin.go:11` | ✅ |
| 45 | `/admin/ad/{id}` | DELETE | `AdmAdController (AdmAdController.java:41)` | `ecampus-crm` | `ad_admin.go:23` | ✅ |
| 46 | `/admin/ad/{id}` | PUT | `AdmAdController (AdmAdController.java:49)` | `ecampus-crm` | `ad_admin.go:40` | ✅ |
| 47 | `/admin/ad/{id}` | GET | `AdmAdController (AdmAdController.java:63)` | `ecampus-crm` | `ad_admin.go:61` | ✅ |
| 48 | `/admin/ad/list` | GET | `AdmAdController (AdmAdController.java:72)` | `ecampus-crm` | `ad_admin.go:79` | ✅ |
| 49 | `/admin/notice` | POST | `AdmNoticeController (AdmNoticeController.java:28)` | `ecampus-crm` | `notice_admin.go:11` | ✅ |
| 50 | `/admin/notice/{id}` | DELETE | `AdmNoticeController (AdmNoticeController.java:34)` | `ecampus-crm` | `notice_admin.go:23` | ✅ |
| 51 | `/admin/notice/{id}` | PUT | `AdmNoticeController (AdmNoticeController.java:42)` | `ecampus-crm` | `notice_admin.go:40` | ✅ |
| 52 | `/admin/notice/{id}` | GET | `AdmNoticeController (AdmNoticeController.java:53)` | `ecampus-crm` | `notice_admin.go:61` | ✅ |
| 53 | `/admin/notice/list` | GET | `AdmNoticeController (AdmNoticeController.java:62)` | `ecampus-crm` | `notice_admin.go:79` | ✅ |
| 54 | `/api/course_color` | POST | `CourseColorController (CourseColorController.java:28)` | `ecampus` | `handler.go:45` | ✅ |
| 55 | `/api/term/list` | GET | `TermController (TermController.java:40)` | `ecampus` | `handler.go:20` | ✅ |
| 56 | `/api/term` | GET | `TermController (TermController.java:48)` | `ecampus` | `handler.go:29` | ✅ |
| 57 | `/admin/term` | POST | `AdmTermController (AdmTermController.java:43)` | `ecampus-crm` | `admin.go:15` | ✅ |
| 58 | `/admin/term/{id}` | DELETE | `AdmTermController (AdmTermController.java:56)` | `ecampus-crm` | `admin.go:28` | ✅ |
| 59 | `/admin/term/cur` | POST | `AdmTermController (AdmTermController.java:68)` | `ecampus-crm` | `admin.go:36` | ✅ |
| 60 | `/api/collection/topic/{topic_id}` | POST | `CollectionController (CollectionController.java:50)` | `ecampus` | `handler.go:162` | ✅ |
| 61 | `/api/collection/topic/{topic_id}` | DELETE | `CollectionController (CollectionController.java:62)` | `ecampus` | `handler.go:170` | ✅ |
| 62 | `/api/collection/collection_topics` | GET | `CollectionController (CollectionController.java:81)` | `ecampus` | `handler.go:178` | ✅ |
| 63 | `/api/comment/{topic_id}` | POST | `CommentController (CommentController.java:74)` | `ecampus` | `handler.go:21` | ✅ |
| 64 | `/api/comment/{topic_id}/{comment_id}` | DELETE | `CommentController (CommentController.java:145)` | `ecampus` | `handler.go:34` | ✅ |
| 65 | `/api/comment/{topic_id}` | GET | `CommentController (CommentController.java:197)` | `ecampus` | `handler.go:42` | ✅ |
| 66 | `/api/comment` | GET | `CommentController (CommentController.java:243)` | `ecampus` | `handler.go:53` | ✅ |
| 67 | `/api/comment/target_user_comments` | GET | `CommentController (CommentController.java:289)` | `ecampus` | `handler.go:63` | ✅ |
| 68 | `/api/comment_like/{comment_id}` | POST | `CommentLikeController (CommentLikeController.java:26)` | `ecampus` | `handler.go:74` | ✅ |
| 69 | `/api/comment_like/{comment_id}` | DELETE | `CommentLikeController (CommentLikeController.java:35)` | `ecampus` | `handler.go:82` | ✅ |
| 70 | `/api/support/{key}` | GET | `FrontendSupportController (FrontendSupportController.java:34)` | `ecampus` | `support.go:9` | ✅ |
| 71 | `/api/support/list` | GET | `FrontendSupportController (FrontendSupportController.java:47)` | `ecampus` | `support.go:18` | ✅ |
| 72 | `/api/report_comment` | POST | `ReportCommentController (ReportCommentController.java:31)` | `ecampus` | `report.go:12` | ✅ |
| 73 | `/admin/sensitive/getAllList` | GET | `SensitiveWordController (SensitiveWordController.java:22)` | `ecampus-crm` | `sensitive_admin.go:9` | ✅ |
| 74 | `/admin/sensitive/getByWord/` | GET | `SensitiveWordController (SensitiveWordController.java:28)` | `ecampus-crm` | `sensitive_admin.go:18` | ⚠️ 路径不一致 |
| 75 | `/admin/sensitive/deleteByWord` | DELETE | `SensitiveWordController (SensitiveWordController.java:40)` | `ecampus-crm` | `sensitive_admin.go:27` | ✅ |
| 76 | `/admin/sensitive/batchDelete` | DELETE | `SensitiveWordController (SensitiveWordController.java:48)` | `ecampus-crm` | `sensitive_admin.go:35` | ✅ |
| 77 | `/admin/sensitive/add` | POST | `SensitiveWordController (SensitiveWordController.java:56)` | `ecampus-crm` | `sensitive_admin.go:47` | ✅ |
| 78 | `/admin/sensitive/batchAdd` | POST | `SensitiveWordController (SensitiveWordController.java:64)` | `ecampus-crm` | `sensitive_admin.go:60` | ✅ |
| 79 | `/admin/sensitive/page` | GET | `SensitiveWordController (SensitiveWordController.java:72)` | `ecampus-crm` | `sensitive_admin.go:72` | ✅ |
| 80 | `/admin/sensitive/search_like` | GET | `SensitiveWordController (SensitiveWordController.java:82)` | `ecampus-crm` | `sensitive_admin.go:82` | ✅ |
| 81 | `/admin/sensitive/update` | PUT | `SensitiveWordController (SensitiveWordController.java:90)` | `ecampus-crm` | `sensitive_admin.go:91` | ✅ |
| 82 | `/api/theme/campus/init` | POST | `ThemeController (ThemeController.java:34)` | `ecampus` | `handler.go:15` | ✅ |
| 83 | `/api/theme/campus` | GET | `ThemeController (ThemeController.java:40)` | `ecampus` | `handler.go:24` | ✅ |
| 84 | `/api/topic` | POST | `TopicController (TopicController.java:96)` | `ecampus` | `handler.go:21` | ✅ |
| 85 | `/api/topic/{id}` | DELETE | `TopicController (TopicController.java:135)` | `ecampus` | `handler.go:34` | ✅ |
| 86 | `/api/topic/{topic_id}` | GET | `TopicController (TopicController.java:173)` | `ecampus` | `handler.go:42` | ✅ |
| 87 | `/api/topic/{topic_id}` | PUT | `TopicController (TopicController.java:182)` | `ecampus` | `handler.go:52` | ✅ |
| 88 | `/api/topic/search` | GET | `TopicController (TopicController.java:231)` | `ecampus` | `handler.go:64` | ✅ |
| 89 | `/api/topic` | GET | `TopicController (TopicController.java:263)` | `ecampus` | `handler.go:91` | ✅ |
| 90 | `/api/topic/theme` | GET | `TopicController (TopicController.java:286)` | `ecampus` | `handler.go:101` | ✅ |
| 91 | `/api/topic/target_user_topics` | GET | `TopicController (TopicController.java:311)` | `ecampus` | `handler.go:111` | ✅ |
| 92 | `/api/topic/follow_topics` | GET | `TopicController (TopicController.java:332)` | `ecampus` | `handler.go:126` | ✅ |
| 93 | `/api/like/topic/{topic_id}` | POST | `TopicLikeController (TopicLikeController.java:33)` | `ecampus` | `handler.go:136` | ✅ |
| 94 | `/api/like/topic/{topic_id}` | DELETE | `TopicLikeController (TopicLikeController.java:55)` | `ecampus` | `handler.go:144` | ✅ |
| 95 | `/api/like/topic` | GET | `TopicLikeController (TopicLikeController.java:73)` | `ecampus` | `handler.go:152` | ✅ |
| 96 | `/api/wx/unlimited/wxa_code` | POST | `WXaCodeController (WXaCodeController.java:26)` | `ecampus` | `handler.go:327` | ✅ |
| 97 | `/admin/comment/{topic_id}/{comment_id}` | DELETE | `AdmCommentController (AdmCommentController.java:25)` | `ecampus-crm` | `admin.go:17` | ✅ |
| 98 | `/admin/support` | POST | `AdmFrontendSupportController (AdmFrontendSupportController.java:35)` | `ecampus-crm` | `support_admin.go:9` | ✅ |
| 99 | `/admin/support` | PUT | `AdmFrontendSupportController (AdmFrontendSupportController.java:42)` | `ecampus-crm` | `support_admin.go:22` | ✅ |
| 101 | `/admin/support/list` | GET | `AdmFrontendSupportController (AdmFrontendSupportController.java:70)` | `ecampus-crm` | `support_admin.go:42` | ✅ |
| 102 | `/admin/merchant_theme` | POST | `AdmMerchantThemeController (AdmMerchantThemeController.java:27)` | `ecampus-crm` | `merchant_admin.go:9` | ✅ |
| 103 | `/admin/merchant_theme/{id}` | DELETE | `AdmMerchantThemeController (AdmMerchantThemeController.java:37)` | `ecampus-crm` | `merchant_admin.go:22` | ✅ |
| 104 | `/admin/merchant_theme/get_all` | GET | `AdmMerchantThemeController (AdmMerchantThemeController.java:45)` | `ecampus-crm` | `merchant_admin.go:30` | ✅ |
| 105 | `/admin/report_comment/{id}` | PUT | `AdmReportCommentController (AdmReportCommentController.java:44)` | `ecampus-crm` | `report_admin.go:10` | ✅ |
| 106 | `/admin/report_comment/list` | GET | `AdmReportCommentController (AdmReportCommentController.java:64)` | `ecampus-crm` | `report_admin.go:22` | ✅ |
| 107 | `/admin/theme/{id}` | PUT | `AdmThemeController (AdmThemeController.java:36)` | `ecampus-crm` | `admin.go:17` | ✅ |
| 108 | `/admin/theme` | GET | `AdmThemeController (AdmThemeController.java:51)` | `ecampus-crm` | `admin.go:30` | ✅ |
| 109 | `/admin/theme/search` | PUT | `AdmThemeController (AdmThemeController.java:62)` | `ecampus-crm` | `admin.go:39` | ✅ |
| 110 | `/admin/theme/suggest` | POST | `AdmThemeController (AdmThemeController.java:72)` | `ecampus-crm` | `admin.go:51` | ✅ |
| 111 | `/admin/theme/campus` | POST | `AdmThemeController (AdmThemeController.java:78)` | `ecampus-crm` | `admin.go:64` | ✅ |
| 112 | `/admin/theme/campus/{themeId}` | DELETE | `AdmThemeController (AdmThemeController.java:88)` | `ecampus-crm` | `admin.go:77` | ✅ |
| 113 | `/admin/topic/{topic_id}` | DELETE | `AdmTopicController (AdmTopicController.java:32)` | `ecampus-crm` | `admin.go:19` | ✅ |
| 114 | `/admin/topic/refresh_suggest` | GET | `AdmTopicController (AdmTopicController.java:40)` | `ecampus-crm` | `admin.go:27` | ✅ |
| 115 | `/api/user/login` | POST | `UserController (UserController.java:45)` | `ecampus` | `handler.go:21` | ✅ |
| 116 | `/api/user` | GET | `UserController (UserController.java:51)` | `ecampus` | `handler.go:124` | ✅ |
| 117 | `/api/user/refresh` | POST | `UserController (UserController.java:72)` | `ecampus` | `handler.go:43` | ✅ |
| 118 | `/api/user` | PUT | `UserController (UserController.java:78)` | `ecampus` | `handler.go:134` | ✅ |
| 119 | `/api/user/authentication` | POST | `UserController (UserController.java:87)` | `ecampus` | `handler.go:148` | ✅ |
| 120 | `/api/user/re_authentication` | POST | `UserController (UserController.java:98)` | `ecampus` | `handler.go:172` | ✅ |
| 121 | `/api/user/del_authentication` | POST | `UserController (UserController.java:110)` | `ecampus` | `handler.go:196` | ✅ |
| 122 | `/api/user/check_login` | POST | `UserController (UserController.java:120)` | `ecampus` | `handler.go:204` | ✅ |
| 123 | `/api/user/get_course_by_weeks` | POST | `UserController (UserController.java:132)` | `ecampus` | `handler.go:228` | ✅ |
| 124 | `/api/user/get_exam` | POST | `UserController (UserController.java:154)` | `ecampus` | `handler.go:256` | ✅ |
| 125 | `/api/user/get_exam_score` | POST | `UserController (UserController.java:176)` | `ecampus` | `handler.go:284` | ✅ |
| 126 | `/api/user/user_profile` | GET | `UserController (UserController.java:205)` | `ecampus` | `handler.go:312` | ✅ |
| 127 | `/api/user/pre_authentication` | PUT | `UserController (UserController.java:211)` | `ecampus` | `handler.go:68` | ✅ |
| 128 | `/api/user/official/certification` | POST | `UserController (UserController.java:220)` | `ecampus` | `handler.go:101` | ✅ |
| 129 | `/api/user/official/login` | POST | `UserController (UserController.java:226)` | `ecampus` | `handler.go:81` | ✅ |
| 130 | `/api/user/identity/anonymous` | POST | `UserController (UserController.java:232)` | `ecampus` | `handler_identity.go:7` | ✅ |
| 131 | `/api/user/identity/anonymous/nickname` | PUT | `UserController (UserController.java:238)` | `ecampus` | `handler_identity.go:16` | ✅ |
| 132 | `/api/user/identity/list` | GET | `UserController (UserController.java:244)` | `ecampus` | `handler_identity.go:28` | ✅ |
| 133 | `/api/user/identity/switch` | POST | `UserController (UserController.java:250)` | `ecampus` | `handler_identity.go:37` | ✅ |
| 134 | `/api/user/nickname/random` | GET | `UserController (UserController.java:256)` | `ecampus` | `handler.go:115` | ✅ |
| 135 | `/api/user/follow` | POST | `UserController (UserController.java:269)` | `ecampus` | `handler_follow.go:11` | ✅ |
| 136 | `/api/user/follow` | DELETE | `UserController (UserController.java:286)` | `ecampus` | `handler_follow.go:24` | ✅ |
| 137 | `/api/user/followers` | GET | `UserController (UserController.java:298)` | `ecampus` | `handler_follow.go:37` | ✅ |
| 138 | `/api/user/followings` | GET | `UserController (UserController.java:319)` | `ecampus` | `handler_follow.go:56` | ✅ |
| 139 | `/api/user/stats` | GET | `UserController (UserController.java:341)` | `ecampus` | `handler_follow.go:75` | ✅ |
| 140 | `/api/user/is_following` | GET | `UserController (UserController.java:353)` | `ecampus` | `handler_follow.go:89` | ✅ |
| 141 | `/admin/user` | POST | `AdminUserController (AdminUserController.java:32)` | `ecampus-crm` | `admin.go:35` | ✅ |
| 142 | `/admin/user/login` | POST | `AdminUserController (AdminUserController.java:37)` | `ecampus-crm` | `admin.go:22` | ✅ |
| 143 | `/admin/user/add` | POST | `AdminUserController (AdminUserController.java:42)` | `ecampus-crm` | `admin.go:47` | ✅ |
| 144 | `/admin/user/{id}` | DELETE | `AdminUserController (AdminUserController.java:47)` | `ecampus-crm` | `admin.go:59` | ✅ |
| 145 | `/admin/user/{id}` | PUT | `AdminUserController (AdminUserController.java:55)` | `ecampus-crm` | `admin.go:76` | ✅ |
| 146 | `/admin/user/{id}` | GET | `AdminUserController (AdminUserController.java:66)` | `ecampus-crm` | `admin.go:97` | ✅ |
| 147 | `/admin/user/list` | GET | `AdminUserController (AdminUserController.java:75)` | `ecampus-crm` | `admin.go:115` | ✅ |
| 148 | `/admin/user/clear` | POST | `AdminUserController (AdminUserController.java:82)` | `ecampus-crm` | `admin.go:126` | ✅ |
| 149 | `/admin/user/course` | POST | `AdminUserController (AdminUserController.java:87)` | `ecampus-crm` | `admin.go:138` | ✅ |
| 150 | `/admin/user/add_black_list` | POST | `AdminUserController (AdminUserController.java:94)` | `ecampus-crm` | `admin.go:158` | ✅ |
| 151 | `/admin/user/del_black_list` | DELETE | `AdminUserController (AdminUserController.java:100)` | `ecampus-crm` | `admin.go:171` | ✅ |
| 152 | `/admin/user/black_list` | GET | `AdminUserController (AdminUserController.java:106)` | `ecampus-crm` | `admin.go:184` | ✅ |
| 153 | `/admin/user/certification/list` | GET | `AdminUserController (AdminUserController.java:112)` | `ecampus-crm` | `admin.go:193` | ✅ |
| 154 | `/admin/user/certification/review` | POST | `AdminUserController (AdminUserController.java:121)` | `ecampus-crm` | `admin.go:204` | ✅ |
| 155 | `/chat` | WS | `WebSocketConfig (WebSocketConfig.java:20)` | `ecampus` | `ws.go:82` | ✅ |
| 156 | `/health` | GET | `—` | `ecampus` | `—` | ⚠️ Go 多余 |
| 157 | `/metrics` | GET | `—` | `ecampus` | `—` | ⚠️ Go 多余 |
| 158 | `/health` | GET | `—` | `ecampus-crm` | `—` | ⚠️ Go 多余 |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致，或 Go 额外暴露了 Java 基线不存在的端点

### 差异清单

#### DIFF-ROUTE-01: `/api/testAop` 活跃端点在 Go 中完全缺失

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `level/src/main/java/com/jb/level/controller/LevelController.java:47-51`
```java
@GetMapping("/testAop")
@ApiIgnore
public Result<?> testAop() {
    String a = "这是/testAop接口";
    return R.data(a);
}
```
- **Go 证据**: `cmd/ecampus/routes.go:128-130`
```go
api.GET("/getUserSignDetail", handlers.Level.GetUserSignDetail)
api.POST("/sign_in", handlers.Level.SignIn)
api.GET("/UserExp", handlers.Level.UserExp)
```
- **模拟场景**:
  - 输入: `GET /api/testAop`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":"这是/testAop接口"}`
  - Go 行为: `HTTP 404`，响应体不是统一 JSON 包装
- **预期行为**: Go 应继续提供 `/api/testAop`，并返回与现网一致的成功 JSON 结构。
- **影响面**: `/api/testAop`

#### DIFF-ROUTE-02: `/api/exp+3/{id}` 活跃端点在 Go 中完全缺失

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `level/src/main/java/com/jb/level/controller/LevelController.java:68-71`
```java
@PostMapping("/exp+3/{id}")
@ApiIgnore
public Result<?> expPlus3(@PathVariable int id) {
    return R.success().msg("经验+3，告辞");
}
```
- **Go 证据**: `cmd/ecampus/routes.go:128-130`
```go
api.GET("/getUserSignDetail", handlers.Level.GetUserSignDetail)
api.POST("/sign_in", handlers.Level.SignIn)
api.GET("/UserExp", handlers.Level.UserExp)
```
- **模拟场景**:
  - 输入: `POST /api/exp+3/1`
  - Java 行为: `{"success":true,"code":200,"msg":"经验+3，告辞","data":null}`
  - Go 行为: `HTTP 404`，响应体不是统一 JSON 包装
- **预期行为**: Go 应继续提供 `/api/exp+3/{id}`，并保持现网返回结构与消息文本。
- **影响面**: `/api/exp+3/{id}`

#### DIFF-ROUTE-03: `/api/event/` 在 Go 中注册成了无尾斜杠版本，POST 会先收到重定向

- **等级**: P2
- **分类**: API契约
- **Java 证据**: `front-event/src/main/java/com/jb/event/controller/EventController.java:34-41`
```java
@ApiOperation("添加埋点数据")
@PostMapping("/")
public Result<?> add(@RequestBody @Validated Event event) {
    event.setUserId(ThreadLocalUtil.getUserId().toString());
    event.setTriggerTime(new Timestamp(System.currentTimeMillis()));
```
- **Go 证据**: `cmd/ecampus/routes.go:150-150`
```go
api.POST("/event", handlers.Event.Add)
```
- **模拟场景**:
  - 输入: `POST /api/event/`，请求体 `{"eventType":"page_view","eventInfo":"home","eventContent":"enter"}`，携带有效 JWT
  - Java 行为: 直接进入处理链，返回 `200` JSON 成功响应，并写入事件数据
  - Go 行为: 先返回 `307 Temporary Redirect`，`Location: /api/event`；本次请求不会直接执行事件写入
- **预期行为**: 对外暴露路径应继续兼容现网的 `/api/event/`。
- **影响面**: `/api/event/`

#### DIFF-ROUTE-04: `/admin/sensitive/getByWord/` 在 Go 中丢了尾斜杠兼容

- **等级**: P2
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:28-35`
```java
@GetMapping("/getByWord/")
public Result<?> getSensitiveWordByWord(@RequestParam("word") String word) {
    if (word == null) {
        return R.fail().msg("参数为NULL，请重试");
    }
```
- **Go 证据**: `cmd/ecampus-crm/routes.go:99-100`
```go
admin.GET("/sensitive/getAllList", handlers.Other.SensitiveGetAllList)
admin.GET("/sensitive/getByWord", handlers.Other.SensitiveGetByWord)
```
- **模拟场景**:
  - 输入: `GET /admin/sensitive/getByWord/?word=spam`
  - Java 行为: 直接命中控制器并返回 JSON
  - Go 行为: 先返回 `301 Moved Permanently` 到 `/admin/sensitive/getByWord?word=spam`
- **预期行为**: 管理端敏感词查询路径应继续兼容现网公开的 `/admin/sensitive/getByWord/`。
- **影响面**: `/admin/sensitive/getByWord/`

### 模块总结

- 活跃端点: 155 个
- Go 已覆盖: 153 个
- P0 差异: 2 个
- P1 差异: 0 个
- P2 差异: 2 个

## 模块：中间件与交叉验证

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/report_comment` | POST | `ReportCommentController.java:31` | `report.go:12` | ✅ |
| 2 | `/api/topic` | POST | `TopicController.java:96` | `handler.go:21` | ✅ |
| 3 | `/api/sign_in` | POST | `LevelController.java:55` | `handler.go:30` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-GLOB-01: Java 对未完成校园认证用户的写接口封禁在 Go 中缺失

- **等级**: P2
- **分类**: 中间件行为
- **Java 证据**: `aop/src/main/java/com/jb/aop/user/AuthPermissionAOP.java:47-68`，`service-base/src/main/java/com/jb/common/result/exceptionHandler/GlobalException.java:24-27`
```java
@Before("isCheck()")
public void doBefore(JoinPoint joinPoint) {
    String method = httpServletRequest.getMethod();
    if(method.equalsIgnoreCase("GET") || method.equalsIgnoreCase("OPTIONS")) {
        return;
    }
    ...
    if(!one.getStuIsCheck()) {
        throw new RuntimeException("当前接口需要进行认证后，方可使用");
    }
}
```
```java
@ExceptionHandler(Exception.class)
public Result<?> globalException(Exception e) {
    return R.fail().msg(e.getMessage());
}
```
- **Go 证据**: `cmd/ecampus/routes.go:60-65`，`internal/other/report.go:12-29`，`internal/other/service_report.go:17-34`
```go
api.Use(
    middleware.JWTAuth(jwtHelper, rds),
    middleware.BlackListCheck(rds),
    middleware.RequestLog(logger),
)
```
```go
report, err := h.svc.CreateReportComment(c.Request.Context(), &ReportComment{
    CommentID:     req.CommentID,
    ReportContent: req.ReportContent,
    ReportUserID:  strconv.FormatInt(middleware.GetUserID(c), 10),
})
```
```go
report.HasHandle = false
res, err := s.mongoDB.Collection("campus_report_comment").InsertOne(ctx, report)
```
- **模拟场景**:
  - 输入: 已登录用户 `userId=42`，数据库中 `campus_user.stuIsCheck=false`；请求 `POST /api/report_comment`，请求体 `{"commentId":"507f1f77bcf86cd799439011","reportContent":"spam"}`，且该评论存在
  - Java 行为: 返回 `{"success":false,"code":400,"msg":"当前接口需要进行认证后，方可使用","data":null}`，`campus_report_comment` 不写入任何新文档
  - Go 行为: 请求继续执行，Mongo 新增 `{"commentId":"507f1f77bcf86cd799439011","reportContent":"spam","reportUserId":"42","hasHandle":false,...}`，接口返回成功 JSON
- **预期行为**: 对未完成校园认证用户，非 GET/OPTIONS 的受限写接口应继续拒绝执行，并且不得产生落库副作用。
- **影响面**: `/api/report_comment` 以及所有未被排除的 `/api/**`、`/admin/**` 非 GET/OPTIONS 写接口

#### DIFF-GLOB-02: Java 的商家专属主题发帖限制在 Go 中缺失

- **等级**: P2
- **分类**: 中间件行为
- **Java 证据**: `aop/src/main/java/com/jb/aop/user/MerchantPermissionAOP.java:35-59`，`theme/src/main/java/com/jb/theme/controller/TopicController.java:96-119`
```java
@Before("execution(* com.jb.theme.controller.TopicController.add(..))")
public void MerchantPostOnly(JoinPoint joinPoint) {
    ...
    if(!Merchant.isMerchant(power)) {
        throw new RuntimeException("当前帖子类型只有商家可以发布");
    }
}
```
```java
@PostMapping
public Result<?> add(@Validated @RequestBody Topic topic) {
    ...
    Topic save = topicService.saveOne(topic);
    topicCheckProducer.produce(checkData);
    return R.data(save);
}
```
- **Go 证据**: `internal/topic/handler.go:21-31`，`internal/topic/service.go:46-96`，`internal/topic/service_helpers.go:41-53`
```go
func (h *Handler) Create(c *gin.Context) {
    var req CreateTopicReq
    ...
    data, err := h.svc.Create(c.Request.Context(), middleware.GetClaims(c), &req)
```
```go
if err := s.ensureThemeExists(ctx, req.ThemeID); err != nil {
    return nil, err
}
...
res, err := s.topicColl().InsertOne(ctx, topic)
...
sendErr := s.producer.SendTopicCheck(ctx, mq.TopicCheckMsg{TopicID: oid.Hex()})
```
```go
err := s.mongoDB.Collection("campus_theme_id").FindOne(ctx, bson.M{"themeId": themeID}).Err()
if err == nil {
    return nil
}
```
- **模拟场景**:
  - 输入: 已登录非商家用户 `userId=42,power=0`，`campus_merchant_theme` 中存在 `themeId="merchant-theme"`；请求 `POST /api/topic`，请求体 `{"themeId":"merchant-theme","title":"促销","content":"全场八折"}` 
  - Java 行为: 在进入 Controller 业务前抛出 `RuntimeException`，返回 `{"success":false,"code":400,"msg":"当前帖子类型只有商家可以发布","data":null}`，既不写入 `campus_topic`，也不发送发帖审核 MQ
  - Go 行为: 正常写入 `campus_topic` 新文档，`hasCheck=false`，并发送 `campus.topic_check` 消息
- **预期行为**: 商家专属主题应继续只允许商家身份发帖，非商家请求不得产生帖子落库和 MQ 副作用。
- **影响面**: `/api/topic` `POST`

#### DIFF-GLOB-03: Java 的 `controller_time` 采集链路在 Go 中完全缺失

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `aop/src/main/java/com/jb/aop/user/ExpAndControllerTimeAop.java:47-90`，`monitor/src/main/java/com/jb/monitor/dao/ControllerTimeDaoImpl.java:16-19`，`monitor/src/main/java/com/jb/monitor/redis/ControllerTimeRedisService.java:21-23`，`level/src/main/java/com/jb/level/scheduleTask/InsertControllerTimeTask.java:30-45`
```java
@Around("pointCut()")
public Object around(ProceedingJoinPoint proceedingJoinPoint) throws Throwable {
    ...
    ControllerTime controllerTime = new ControllerTime(...);
    controllerTimeDao.insertControllerTime(controllerTime);
}
```
```java
public void insertControllerTime(ControllerTime controllerTime) {
    controllerTimeRedisService.insertOne(controllerTime);
}
```
```java
public void insertOne(ControllerTime controllerTime) {
    redisTemplate.opsForList().leftPush(getCTIME_KEY(controllerTime.getController()), controllerTime);
}
```
```java
@Scheduled(cron = "* 0/10 * * * ?")
public void insertAllControllerTimeFromRedis() {
    List<ControllerTime> controllerTimeList = controllerTimeRedisService.getAndDeleteAll();
    ...
    controllerTimeMapper.saveBatch(list);
}
```
- **Go 证据**: `internal/middleware/log.go:15-45`，`internal/cron/scheduler.go:23-88`
```go
return func(c *gin.Context) {
    start := time.Now()
    ...
    logger.Info("http request", fields...)
}
```
```go
return &Scheduler{
    cron:      robcron.New(robcron.WithSeconds()),
    suggest:   NewSuggestJob(...),
    event:     NewEventFlushJob(...),
    metrics:   NewMetricsJob(...),
    expDetail: NewExpFlushJob(...),
}
```
- **模拟场景**:
  - 输入: 任一成功命中的控制器请求，例如 `POST /api/report_comment`
  - Java 行为: 先向 Redis 写入 `campus:controllerTime:*` 列表项；10 分钟批任务再把这些记录刷入 MySQL `controller_time`
  - Go 行为: 只输出应用日志；既不写 Redis，也不写 MySQL `controller_time`
- **预期行为**: Go 应继续产出现网依赖的 `controller_time` 数据写入结果，而不是仅保留日志。
- **影响面**: 所有 Controller / Handler 请求链路，以及依赖 `controller_time` 表的数据消费方

### 核验结论

- `Topic 创建 -> MQ -> 搜索索引` 链路两端都闭环存在。Java 通过 `TopicController.add -> TopicCheckProducer -> TopicCheckConsumer -> AddTopicSearchProducer` 完成；Go 通过 `topic.Service.Create -> mq.SendTopicCheck -> handleTopicCheck -> SendAddTopicSearch -> handleTopicSearchAdd` 完成。按仓库代码未发现本轮新增差异。
- `Comment 创建 -> MQ -> 通知 + commentCount 更新` 链路两端都闭环存在。Java `AddCommentConsumer` 与 Go `handleCommentAdd` 都会在审核通过后更新帖子/根评论计数，并发出通知消息。按仓库代码未发现本轮新增差异。
- 用户被拉黑后，Java 与 Go 都只阻断后续访问，不会追溯修改该用户既有 topic/comment 的可见性条件；仓库内未发现这条链路的跨语言偏差。
- 经验值奖励依赖 `aop_permission` 运行时数据。仓库只有表结构，没有初始化内容，因此“哪些接口应加经验值”本轮无法构造出可验证的必然差异。

### 模块总结

- 活跃端点: 3 个
- Go 已覆盖: 3 个
- P0 差异: 0 个
- P1 差异: 1 个
- P2 差异: 2 个

## 模块：配置文件比对

### 活跃 API 端点清单

本模块不涉及独立 API 端点。

### 差异清单

按当前代码与启动路径，本模块无可报告差异。

### 核验结论

- `configs/ecampus/application.yml` / `application-dev.yml` 已提供用户端服务实际启动所需的 `server/mysql/mongo/redis/rabbitmq/jwt/cos/wx/custom/jw/encryption/admin/logging`。
- `configs/ecampus-crm/application.yml` / `application-dev.yml` 提供了管理端实际启动所需的 `server/mysql/mongo/redis/jwt/encryption/admin/logging`。
- `ecampus-crm` 虽然没有单独配置 `rabbitmq/cos/wx/jw/custom` 段，但 `cmd/ecampus-crm/main.go` 当前不初始化 RabbitMQ；其已注册的管理端路由也不会在无这些配置的情况下触发必须读取这些配置的路径，或已有默认兜底，因此未计为独立部署必现问题。

### 模块总结

- 活跃端点: 0 个
- Go 已覆盖: 0 个
- P0 差异: 0 个
- P1 差异: 0 个
- P2 差异: 0 个
