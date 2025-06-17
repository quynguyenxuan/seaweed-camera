# Tasks Description

Dự án chưa có nhiệm vụ lặp lại được tài liệu hóa. Dưới đây là ví dụ về các nhiệm vụ có thể xảy ra dựa trên phân tích codebase:

- **Thêm hỗ trợ cho mô hình lưu trữ mới**:
  - **Files cần chỉnh sửa**: [`weed/filer/rocksdb/rocksdb_store.go`](weed/filer/rocksdb/rocksdb_store.go), [`weed/filer/filerstore.go`](weed/filer/filerstore.go).
  - **Các bước**: 1. Cập nhật cấu hình trong RocksDB để hỗ trợ key mới. 2. Thêm logic trong filerstore để xử lý dữ liệu mới. 3. Kiểm tra và kiểm tra lại qua codebase_search.
  - **Lưu ý**: Đảm bảo tính nhất quán với master server.

- **Gỡ lỗi prefix search trong RocksDB**:
  - **Files cần chỉnh sửa**: [`weed/filer/rocksdb/rocksdb_store.go`](weed/filer/rocksdb/rocksdb_store.go).
  - **Các bước**: 1. Kiểm tra hàm enumerate. 2. Thêm log debug. 3. Kiểm tra lại với codebase_search.
  - **Lưu ý**: Sao lưu dữ liệu trước khi thay đổi.