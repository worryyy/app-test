# Backend API Endpoints

This document is generated from the current frontend source scan on 2026-04-10.

## Base URL Rules

The project uses `localStorage.selected_environment` as the runtime backend base URL.

- Default: `https://fangfangfang.top`
- Dev: `https://dev.fangfangfang.top`
- Local: `http://localhost:8080`

Related code:

- `src/api/axios.js`
- `src/pages/Dashboard/Dashboard.jsx`
- `src/layouts/DashboardLayout.jsx`

Note:

- The README still mentions `http://43.136.50.116:8080/api`, but the current code does not use that as the active base URL.
- The monitor page uses Grafana page URLs, not backend business APIs, so it is not included here.

## User And Auth

| Method | Production URL | Dev URL | Local URL | Notes |
| --- | --- | --- | --- | --- |
| POST | `https://fangfangfang.top/admin/user/login` | `https://dev.fangfangfang.top/admin/user/login` | `http://localhost:8080/admin/user/login` | Login |
| POST | `https://fangfangfang.top/admin/user/refresh` | `https://dev.fangfangfang.top/admin/user/refresh` | `http://localhost:8080/admin/user/refresh` | Admin refresh token, body `{ "refresh_token": "..." }` |
| POST | `https://fangfangfang.top/admin/user/logout` | `https://dev.fangfangfang.top/admin/user/logout` | `http://localhost:8080/admin/user/logout` | Admin logout, requires admin access token |
| GET | `https://fangfangfang.top/admin/user/list` | `https://dev.fangfangfang.top/admin/user/list` | `http://localhost:8080/admin/user/list` | User list, supports `page`, `size`, `id`, `stu_num`, `nickname`; legacy `nick_name` is still accepted |
| POST | `https://fangfangfang.top/admin/user` | `https://dev.fangfangfang.top/admin/user` | `http://localhost:8080/admin/user` | Create user |
| PUT | `https://fangfangfang.top/admin/user/{id}` | `https://dev.fangfangfang.top/admin/user/{id}` | `http://localhost:8080/admin/user/{id}` | Update user |
| PUT | `https://fangfangfang.top/admin/user/pre_authentication?user_id={user_id}&nick_name={nick_name}&pwd={pwd}` | `https://dev.fangfangfang.top/admin/user/pre_authentication?user_id={user_id}&nick_name={nick_name}&pwd={pwd}` | `http://localhost:8080/admin/user/pre_authentication?user_id={user_id}&nick_name={nick_name}&pwd={pwd}` | Deprecated compatibility endpoint for single pre-authentication |
| PUT | `https://fangfangfang.top/admin/user/pre_authentication/batch` | `https://dev.fangfangfang.top/admin/user/pre_authentication/batch` | `http://localhost:8080/admin/user/pre_authentication/batch` | Batch pre-authentication, body `{ "password": "...", "items": [{ "userId": 1, "nickName": "..." }] }` |
| POST | `https://fangfangfang.top/admin/user/clear` | `https://dev.fangfangfang.top/admin/user/clear` | `http://localhost:8080/admin/user/clear` | Clear auth info, body like `{ "userId": 123 }` |

## Topic

| Method | Production URL | Dev URL | Local URL | Notes |
| --- | --- | --- | --- | --- |
| GET | `https://fangfangfang.top/admin/topic` | `https://dev.fangfangfang.top/admin/topic` | `http://localhost:8080/admin/topic` | Topic list, only `page` and `size` are required |
| POST | `https://fangfangfang.top/admin/topic` | `https://dev.fangfangfang.top/admin/topic` | `http://localhost:8080/admin/topic` | Create topic |
| PATCH | `https://fangfangfang.top/admin/topic/{topic_id}` | `https://dev.fangfangfang.top/admin/topic/{topic_id}` | `http://localhost:8080/admin/topic/{topic_id}` | Update topic |
| DELETE | `https://fangfangfang.top/admin/topic/{topic_id}` | `https://dev.fangfangfang.top/admin/topic/{topic_id}` | `http://localhost:8080/admin/topic/{topic_id}` | Delete topic |
| DELETE | `https://fangfangfang.top/admin/topic/{topic_id}/comment/{comment_id}` | `https://dev.fangfangfang.top/admin/topic/{topic_id}/comment/{comment_id}` | `http://localhost:8080/admin/topic/{topic_id}/comment/{comment_id}` | Delete any comment under a topic with admin permission |
## Term

| Method | Production URL | Dev URL | Local URL | Notes |
| --- | --- | --- | --- | --- |
| GET | `https://fangfangfang.top/admin/term/list` | `https://dev.fangfangfang.top/admin/term/list` | `http://localhost:8080/admin/term/list` | Term list |
| GET | `https://fangfangfang.top/admin/term` | `https://dev.fangfangfang.top/admin/term` | `http://localhost:8080/admin/term` | Current term |
| POST | `https://fangfangfang.top/admin/term` | `https://dev.fangfangfang.top/admin/term` | `http://localhost:8080/admin/term` | Create term |
| DELETE | `https://fangfangfang.top/admin/term/{id}` | `https://dev.fangfangfang.top/admin/term/{id}` | `http://localhost:8080/admin/term/{id}` | Delete term |
| POST | `https://fangfangfang.top/admin/term/cur` | `https://dev.fangfangfang.top/admin/term/cur` | `http://localhost:8080/admin/term/cur` | Set current term, body like `{ "termId": 1 }` |

## Sensitive Word

| Method | Production URL | Dev URL | Local URL | Notes |
| --- | --- | --- | --- | --- |
| GET | `https://fangfangfang.top/admin/sensitive/page` | `https://dev.fangfangfang.top/admin/sensitive/page` | `http://localhost:8080/admin/sensitive/page` | Paged list, commonly with `page`, `size` |
| POST | `https://fangfangfang.top/admin/sensitive/add?word={word}` | `https://dev.fangfangfang.top/admin/sensitive/add?word={word}` | `http://localhost:8080/admin/sensitive/add?word={word}` | Add sensitive word |
| DELETE | `https://fangfangfang.top/admin/sensitive/deleteByWord?word={word}` | `https://dev.fangfangfang.top/admin/sensitive/deleteByWord?word={word}` | `http://localhost:8080/admin/sensitive/deleteByWord?word={word}` | Delete by word |
| GET | `https://fangfangfang.top/admin/sensitive/search_like?word={word}` | `https://dev.fangfangfang.top/admin/sensitive/search_like?word={word}` | `http://localhost:8080/admin/sensitive/search_like?word={word}` | Search by word |

## Source Files Used For The Scan

- `src/api/services.js`
- `src/api/axios.js`
- `src/pages/LoginPage/LoginPage.jsx`
- `src/pages/PreAuthPage/PreAuthPage.jsx`
- `src/pages/TopicManagePage/TopicManagePage.jsx`
- `src/pages/SemesterManagePage/SemesterManagePage.jsx`
- `src/pages/SensitiveWordManagePage/SensitiveWordManagePage.jsx`

## Extra Finding

The admin backend contract now expects `POST /admin/user/refresh`; if the admin frontend still calls `/api/user/refresh`, that call path needs to be migrated separately.
