# Tech Description

- **Công nghệ sử dụng**: Dự án được xây dựng bằng Go lang, với các thành phần chính như SeaweedFS cho lưu trữ phân tán, RocksDB cho cơ sở dữ liệu cục bộ, và hỗ trợ S3 cho tích hợp cloud storage.
- **Thiết lập phát triển**: Sử dụng Go modules cho quản lý dependencies, VSCode cho môi trường phát triển, và công cụ như `go build` để biên dịch. Cấu hình RocksDB yêu cầu cài đặt thủ công và tích hợp qua wrapper C.
- **Ràng buộc kỹ thuật**: Hệ thống phải xử lý lưu trữ lớn với tính nhất quán cao, hỗ trợ replication và load balancing. RocksDB được sử dụng cho tốc độ truy vấn nhanh, nhưng cần quản lý bộ nhớ cẩn thận để tránh rò rỉ.
- **Dependencies**: Các package chính bao gồm `gorocksdb` cho RocksDB, `github.com/seaweedfs/seaweedfs` cho lõi dự án, và các thư viện AWS SDK cho S3 compatibility.
- **Mẫu sử dụng công cụ**: Sử dụng codebase_search để phân tích cấu trúc dự án, read_file để kiểm tra file, và write_to_file để cập nhật Memory Bank. Trong phát triển, sử dụng lệnh như `go run` để kiểm tra code.