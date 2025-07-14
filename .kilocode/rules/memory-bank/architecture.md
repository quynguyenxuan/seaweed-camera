# Architecture Description

SeaweedFS sử dụng mô hình kiến trúc phân tán với các thành phần chính sau:

- **Master Server**: Quản lý metadata, bao gồm vị trí của các volume và trạng thái của hệ thống. Nó xử lý các yêu cầu từ filer và đảm bảo tính nhất quán.
- **Filer Server**: Xử lý các hoạt động file system như tạo, đọc, ghi, và xóa file. Nó tương tác với master để lấy metadata và với volume server để lưu trữ dữ liệu.
- **Volume Server**: Lưu trữ dữ liệu thực tế dưới dạng volume, cho phép mở rộng ngang và replication để tăng độ tin cậy.
    - **Cơ chế ghi file**:
        1.  **Tiếp nhận yêu cầu**: Yêu cầu ghi file (dưới dạng một `needle`) được gửi đến `Volume Server`.
        2.  **Xử lý tại Store**: Hàm `WriteVolumeNeedle` trong `Store` tìm kiếm `Volume` đích.
        3.  **Chuyển tiếp đến Volume**: Nếu `Volume` tồn tại và có thể ghi, yêu cầu được chuyển đến hàm `writeNeedle2` của `Volume`.
        4.  **Đồng bộ hoặc bất đồng bộ**: Nếu không yêu cầu `fsync`, việc ghi được thực hiện đồng bộ. Nếu yêu cầu `fsync`, `writeNeedle2` sẽ đưa yêu cầu vào một kênh bất đồng bộ (`asyncRequestsChan`).
        5.  **Worker xử lý**: Một goroutine worker liên tục đọc các yêu cầu từ `asyncRequestsChan`.
        6.  **Ghi dữ liệu thực tế (`doWriteRequest`)**: Cả luồng đồng bộ và worker đều gọi hàm `doWriteRequest`. Hàm này nối dữ liệu của `needle` vào file `.dat` của volume và cập nhật thông tin vị trí, kích thước của `needle` vào `needle map`.
        7.  **Đảm bảo tính bền vững**: Nếu được yêu cầu `fsync`, worker sẽ gọi `v.DataBackend.Sync()` để đảm bảo dữ liệu được ghi vật lý xuống đĩa.
    - **Tận dụng CPU đa luồng và song song hóa ghi**: Để tận dụng tối đa CPU đa luồng, cần tạo nhiều `Volume` trên mỗi `DiskLocation` (hoặc nhiều `DiskLocation` trên các ổ đĩa vật lý khác nhau). Mỗi `Volume` có một goroutine `startWorker` riêng để xử lý các yêu cầu ghi, cho phép các thao tác ghi diễn ra song song.

Các thành phần chính trong codebase bao gồm:
- Các file trong thư mục `weed/filer/`: Xử lý lưu trữ và truy vấn dữ liệu, như sử dụng RocksDB cho cơ sở dữ liệu cục bộ.

- **Cơ chế cấp phát đĩa**:
    - **`DiskLocation`**: Quản lý các thư mục lưu trữ vật lý, theo dõi dung lượng trống và số lượng volume tối đa.
    - **`Volume`**: Đại diện cho một đơn vị lưu trữ logic. File `.dat` của volume có thể được cấp phát trước (`preallocate`) để tối ưu hiệu suất ghi tuần tự.
    - **Điểm tắc nghén tiềm năng**: Việc kiểm tra dung lượng đĩa định kỳ và chính sách cấp phát trước có thể ảnh hưởng đến hiệu suất ban đầu hoặc khi tạo volume mới.

Mối quan hệ: Master quản lý tổng thể, Filer xử lý logic file, và Volume lưu trữ dữ liệu, cho phép hệ thống mở rộng dễ dàng.