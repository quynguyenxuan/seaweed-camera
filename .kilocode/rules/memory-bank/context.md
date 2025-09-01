# Current Context

Dự án đang sử dụng SeaweedFS version 3.96, một dự án Go lang dành cho lưu trữ S3. Các cuộc thảo luận gần đây liên quan đến việc gỡ lỗi prefix search trong RocksDB và cấu hình S3. Memory Bank hiện chỉ có brief.md, và chúng ta đang tạo các file mới để lưu trữ ngữ cảnh.

Các thay đổi gần đây đã được thực hiện để tối ưu hóa hiệu suất ghi vào Volume Server:
- **Tăng kích thước buffer `fsync`**: Đã tăng ngưỡng kích thước buffer trước khi thực hiện `fsync` từ 4MB lên 16MB trong `weed/storage/volume_write.go` để giảm tần suất ghi vật lý xuống đĩa.
- **Tăng dung lượng `asyncRequestsChan`**: Đã tăng dung lượng của channel `asyncRequestsChan` từ 128 lên 256 trong `weed/storage/volume.go` để cho phép nhiều yêu cầu ghi bất đồng bộ được đệm và xử lý đồng thời hơn.

Phân tích hiệu suất ghi (hiện đạt 1GB/s) cho thấy các điểm tắc nghẽn tiềm năng tiếp theo có thể liên quan đến hiệu suất ổ đĩa vật lý (đã xác nhận là SSD NVMe), CPU và Garbage Collection của Go, băng thông mạng, overhead hệ thống file, và overhead logic ứng dụng (ví dụ: kiểm tra trùng lặp).

Việc sử dụng `NeedleMapKindMemory` cho thấy `needle map` không phải là nguyên nhân chính gây tắc nghẽn I/O. Cơ chế cấp phát đĩa (bao gồm `DiskLocation` và `preallocate`) cũng đã được phân tích và không phải là điểm tắc nghẽn lớn với SSD NVMe.

**Trạng thái hiện tại**: Đã tối ưu hóa hiệu suất ghi. Đã tạo nhiều `DiskLocation` nhưng tốc độ ghi không cải thiện, cho thấy vấn đề có thể nằm ở việc phân phối tải ghi trên các `Volume` để tận dụng CPU đa luồng.
**Thay đổi gần đây**: Tăng kích thước buffer `fsync` và dung lượng `asyncRequestsChan`.
**Bước tiếp theo**: Để tận dụng tối đa CPU đa luồng, giải pháp chính là đảm bảo các yêu cầu ghi được phân phối rộng rãi trên nhiều `Volume` khác nhau. Điều này có thể đạt được bằng cách tăng số lượng `Volume` đang hoạt động và kiểm tra/tăng mức độ song song hóa của ứng dụng client. Giám sát phân phối tải ghi sẽ giúp xác định hiệu quả của giải pháp này.

**Thông tin bổ sung**: 
- Filer Server được khởi tạo trong hàm `NewFilerServer` trong tệp `weed/server/filer_server.go`. Hàm này được gọi từ hàm `startFiler` trong tệp `weed/command/filer.go`.