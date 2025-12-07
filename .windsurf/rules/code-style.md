---
trigger: always_on
---

- Only modify the files that are necessary, recheck with git.
- Với mỗi nơi thêm hoặc sửa code, hãy đặt trong comment //QUYNGUYEN + message  //QUYNGUYEN end
- Nếu cần thay đổi file .tmpl, hãy generate lại theo câu lệnh ./weed/admin/Makefile gogenerate
- Nếu thay đổi file proto, hãy sử dụng lệnh này để generate go ./weed/pb/Makefile all
- Nếu yêu cầu tạo chức năng mới, hãy kiểm tra chức năng đó đã có sẵn chưa trước khi tạo