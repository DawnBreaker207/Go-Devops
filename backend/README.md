# BackEnd-CP

API service của **Cinema Project**, viết bằng Go + Gin + GORM + PostgreSQL theo clean architecture
(`handler → service → repository`).

## Yêu cầu môi trường

| Thành phần | Phiên bản |
| ---------- | --------- |
| Go | >= 1.26 |
| PostgreSQL | >= 14 (khuyến nghị 16) |
| Docker | tuỳ chọn, để chạy `docker compose` |
| golang-migrate | tuỳ chọn: `brew install golang-migrate` |
| swag | tuỳ chọn: `go install github.com/swaggo/swag/cmd/swag@latest` |

## Chạy dev

```bash
cp .env.example .env                       # sửa DATABASE_* và JWT_* cho phù hợp
openssl rand -hex 32                       # sinh secret cho JWT_ACCESS_SECRET / JWT_REFRESH_SECRET

docker compose up -d postgres            # hoặc dùng Postgres sẵn có
docker compose up -d redis                # tuỳ chọn: bật read cache (REDIS_ADDR đang trống = không cache)
make run                                   # http://localhost:8080
```

Chạy toàn bộ bằng Docker (postgres + rabbitmq + redis + migrate + backend):

```bash
make docker-up      # local: đặt APP_ENV=development (và JWT_*/PAYMENT_PROVIDERS_MOCK_SECRET) trong .env,
                    # vì mặc định compose chạy production và từ chối secret trống
make docker-down
```

Lần đầu chạy ở môi trường khác `production`, service tự tạo tài khoản quản trị khi bảng `users` còn rỗng:

```
email:    admin@cinema.local      (APP_ADMIN_EMAIL)
password: admin123                (APP_ADMIN_PASSWORD)
```

Đổi mật khẩu ngay sau lần đăng nhập đầu tiên. Ở `APP_ENV=production` phần seed này không chạy.

## Scripts

| Lệnh | Mô tả |
| ---- | ----- |
| `make run` | Chạy server |
| `make build` | Build binary vào `bin/backend-cp` |
| `make test` | `go test ./...` (cần Postgres + RabbitMQ đang chạy) |
| `make test-system` | `docker compose up -d postgres rabbitmq` rồi `go test ./... -count=1` |
| `make chaos` | Chạy các test race/concurrency (`Race` / `Stress` / `Concurrent` trong tên) trong container `golang:1.26` với `-race` |
| `make load` | Chạy riêng `TestStress_NoSeatSoldTwice` với `-timeout 30m` |
| `make test-race` | Chạy toàn bộ test với `-race` trong container `golang:1.26` (race detector cần gcc) |
| `make lint` | `go vet ./...` |
| `make fmt` / `make tidy` | Format source / dọn `go.mod` |
| `make swag` | Sinh lại swagger vào `docs/` |
| `make migrate-up` / `make migrate-down` | Chạy / rollback migration (schema chỉ đến từ `migrations/schema`, gom theo module — xem `migrations/README.md`) |
| `make migrate-db-reset` | Xoá và dựng lại DB dev từ migration + seed |
| `make docker-up` / `make docker-down` | Docker compose: postgres, rabbitmq, redis, service `migrate` chạy migration rồi mới bật backend. Compose mặc định `APP_ENV=production` — chạy local thì đặt `APP_ENV=development` trong `.env`, hoặc cung cấp secret thật |

## Biến môi trường

Thứ tự ưu tiên: **biến môi trường → `.env` → `config.yaml` → default trong code**.

| Biến | Mặc định | Mô tả |
| ---- | -------- | ----- |
| `APP_ENV` | `development` | `production` sẽ bật gin release mode, tắt `/swagger`, tắt seed admin |
| `APP_LOG_LEVEL` | `debug` | `debug` / `info` / `warn` / `error` |
| `APP_ADMIN_EMAIL` / `APP_ADMIN_PASSWORD` | `admin@cinema.local` / `admin123` | Tài khoản seed lần đầu |
| `SERVER_PORT` | `8080` | Cổng HTTP |
| `SERVER_SHUTDOWN_TIMEOUT` | `10s` | Thời gian chờ khi graceful shutdown |
| `DATABASE_HOST` / `_PORT` / `_USER` / `_PASSWORD` / `_NAME` | `localhost` / `5432` / `postgres` / `postgres` / `cinema` | Kết nối Postgres |
| `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB` / `REDIS_TTL` | *(trống)* / *(trống)* / `0` / `5m` | Cache đọc phim công khai. Địa chỉ trống = tắt cache |
| `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` | *(bắt buộc)* | Hai secret phải khác nhau, service từ chối khởi động nếu trống hoặc trùng |
| `JWT_ACCESS_TTL` / `JWT_REFRESH_TTL` | `15m` / `168h` | Hạn của access / refresh token |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Danh sách origin, phân tách bằng dấu phẩy |
| `SERVER_TRUSTED_PROXIES` | *(trống)* | IP/CIDR của reverse proxy được tin `X-Forwarded-For`; trống thì IP client là địa chỉ TCP |
| `PAYMENT_LATE_CAPTURE_WINDOW` | `24h` | Thời gian kiểm lại lượt thanh toán đã bỏ để bắt tiền về muộn (≥ 1h) |
| `PAYMENT_PROVIDERS_MOCK_ALLOW_IN_PRODUCTION` | `false` | Cổng mock không thu tiền thật: chỉ chạy ở production khi bật cờ này |

## API

Base path: `/api/v1`. Swagger UI: <http://localhost:8080/swagger/index.html> (tắt ở production).

| Method | Endpoint | Quyền |
| ------ | -------- | ----- |
| `GET` | `/health` | công khai |
| `POST` | `/api/v1/auth/register` | công khai |
| `POST` | `/api/v1/auth/login` | công khai |
| `POST` | `/api/v1/auth/refresh` | công khai |
| `GET` | `/api/v1/users/me` | đã đăng nhập |
| `GET` | `/api/v1/movies?page=&page_size=&search=` | đã đăng nhập |
| `GET` | `/api/v1/movies/{id}` | đã đăng nhập |
| `POST` `PUT` `DELETE` | `/api/v1/movies` `/api/v1/movies/{id}` | `admin`, `staff` |

Mọi response đều theo khung chung:

```jsonc
// thành công
{ "code": 0, "message": "success", "data": { } }

// danh sách có phân trang
{ "code": 0, "message": "success",
  "data": { "items": [], "meta": { "page": 1, "page_size": 10, "total": 42, "total_pages": 5 } } }

// lỗi
{ "code": 40001, "message": "validation failed", "details": { "Email": "must be a valid email address" } }
```

## Cấu trúc thư mục

```
cmd/server/main.go      # wiring + graceful shutdown
internal/
├── config/             # viper: config.yaml + .env + biến môi trường
├── database/           # kết nối GORM, seed admin, health check
├── dto/                # request/response struct + binding rule
├── models/             # entity GORM
├── repository/         # interface + implement, chỉ chạm DB
├── service/            # business logic
├── handlers/           # gin handler: parse request, gọi service, trả response
├── middleware/         # RequestID, Recovery, Logger, CORS, Auth, RequireRoles
└── router/             # khai báo route
pkg/
├── errors/             # AppError + mã lỗi nghiệp vụ
├── jwt/                # phát hành / xác thực access & refresh token
├── logger/             # zap
└── response/           # chuẩn hoá { code, message, data }
docs/                   # swagger sinh tự động (make swag)
migrations/             # SQL cho golang-migrate
```

## Quy ước code

- **Handler không chứa logic nghiệp vụ**: chỉ bind request, gọi service, trả `response.OK/Created/List/Error`.
- **Service trả về `*apperrors.AppError`**; `response.Error` tự map ra HTTP status + mã lỗi, log riêng nhóm 5xx.
- **Repository chỉ nhận/trả model**, không biết gì về `gin.Context`; luôn truyền `context.Context` xuống DB.
- Lỗi validate của `go-playground/validator` được gom thành `details` theo từng field.
- Mật khẩu hash bằng `bcrypt`; response không bao giờ chứa field `password` (tag `json:"-"`).
- Đăng nhập sai không phân biệt "email không tồn tại" và "sai mật khẩu" — cùng trả `40100`.
