# FrontEnd-CP

Giao diện quản trị của **Cinema Project**, xây dựng trên React 18 + TypeScript + Vite + Ant Design v5.

## Yêu cầu môi trường

| Thành phần | Phiên bản                                                                          |
| ---------- | ---------------------------------------------------------------------------------- |
| Node.js    | >= 20 (khuyến nghị 22 LTS)                                                         |
| npm        | >= 10                                                                              |
| BackEnd-CP | chạy tại `http://localhost:8080`, **đã chạy `make migrate` + `make migrate-seed`** |

## Cài đặt & chạy dev

```bash
npm install
cp .env.example .env      # chỉnh VITE_API_BASE_URL nếu cần
npm run dev               # http://localhost:3000
```

Trang chủ lấy toàn bộ nội dung từ backend. Nếu banner trống, lưới phim trống hoặc
khối trailer không hiện, gần như chắc chắn là **backend chưa chạy seed** chứ không
phải lỗi giao diện — chạy `make migrate-seed` bên `BackEnd-CP` rồi tải lại. Lịch
chiếu cũ đi (danh sách suất rỗng) cũng chữa bằng đúng lệnh đó.

## Scripts

| Lệnh                                  | Mô tả                                                |
| ------------------------------------- | ---------------------------------------------------- |
| `npm run dev`                         | Chạy dev server (port 3000)                          |
| `npm run build`                       | Type-check (`tsc -b`) + build production vào `dist/` |
| `npm run preview`                     | Xem thử bản build                                    |
| `npm run lint` / `npm run lint:fix`   | ESLint                                               |
| `npm run format`                      | Prettier                                             |
| `npm run test` / `npm run test:watch` | Vitest                                               |

## Biến môi trường

| Biến                | Mặc định                       | Mô tả                   |
| ------------------- | ------------------------------ | ----------------------- |
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1` | Base URL của BackEnd-CP |
| `VITE_APP_NAME`     | `Cinema Project`               | Tên ứng dụng hiển thị   |

Chỉ biến bắt đầu bằng `VITE_` mới được expose ra client — không đặt secret ở đây.

## Cấu trúc thư mục

```
src/
├── api/            # axios instance + interceptor, service theo domain
│   ├── client.ts   # gắn Bearer token, auto refresh khi 401, chuẩn hoá lỗi
│   ├── auth.api.ts
│   └── movie.api.ts
├── assets/
├── components/     # Loading, ErrorBoundary, PageHeader
├── features/       # chia theo nghiệp vụ
│   ├── auth/       # LoginPage + test
│   ├── dashboard/
│   ├── movie/      # MoviesPage, MovieFormModal, hooks react-query
│   ├── showtime/
│   └── booking/
├── hooks/
├── layouts/        # MainLayout (Sider + Header + Breadcrumb), AuthLayout
├── locales/        # i18n vi/en
├── pages/          # NotFoundPage
├── routes/         # route config, paths, ProtectedRoute
├── stores/         # zustand: authStore, appStore
├── types/          # kiểu dữ liệu dùng chung
├── utils/          # format, storage
├── theme.ts        # antd design token, light/dark
├── App.tsx
└── main.tsx
```

## Kiến trúc & quy ước

- **Server state** dùng `@tanstack/react-query` (`features/*/hooks`), **global state** dùng `zustand` (`stores/`). Không nhét dữ liệu API vào zustand.
- Mọi request đi qua `apiClient`; response backend dạng `{ code, message, data }` được bóc bằng `unwrap()` nên service chỉ trả về `data`.
- Khi access token hết hạn, interceptor tự gọi `/auth/refresh` một lần và xếp hàng các request còn lại; thất bại thì bắn event `cp:unauthorized` để `App.tsx` đưa về `/login`.
- Route được lazy load, bọc trong `ProtectedRoute`; alias `@/` trỏ tới `src/`.
- Theme sáng/tối và ngôn ngữ lưu ở `localStorage`, đổi ngay trên Header.

## Docker

```bash
docker build -t frontend-cp .
docker run --rm -p 3000:80 frontend-cp
```
