# Architecture Description

SeaweedFS sử dụng mô hình kiến trúc phân tán với các thành phần chính sau:

- **Master Server**: Quản lý metadata, bao gồm vị trí của các volume và trạng thái của hệ thống. Nó xử lý các yêu cầu từ filer và đảm bảo tính nhất quán.
- **Filer Server**: Xử lý các hoạt động file system như tạo, đọc, ghi, và xóa file. Nó tương tác với master để lấy metadata và với volume server để lưu trữ dữ liệu.
- **Volume Server**: Lưu trữ dữ liệu thực tế dưới dạng volume, cho phép mở rộng ngang và replication để tăng độ tin cậy.

Các thành phần chính trong codebase bao gồm:
- [`weed/server/master_ui/master.html`](weed/server/master_ui/master.html): Giao diện cho master server.
- [`weed/server/filer_ui/filer.html`](weed/server/filer_ui/filer.html): Giao diện cho filer server.
- Các file trong thư mục `weed/filer/`: Xử lý lưu trữ và truy vấn dữ liệu, như sử dụng RocksDB cho cơ sở dữ liệu cục bộ.

Mối quan hệ: Master quản lý tổng thể, Filer xử lý logic file, và Volume lưu trữ dữ liệu, cho phép hệ thống mở rộng dễ dàng.