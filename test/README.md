# 邮箱服务器性能测试

本目录包含邮箱服务器的性能测试脚本和工具。

## 测试项目

### 1. API性能测试
- 邮箱列表API响应时间
- 邮件发送API吞吐量
- 认证API并发测试

### 2. SMTP性能测试
- SMTP连接性能
- 邮件发送速度测试
- 并发连接测试

### 3. IMAP性能测试
- IMAP连接性能
- 邮件读取速度测试
- 大量邮件处理测试

### 4. 系统资源测试
- 内存使用情况
- CPU占用率
- 磁盘I/O性能

## 测试工具

- `api_test.js` - API接口性能测试
- `smtp_test.go` - SMTP协议性能测试
- `imap_test.go` - IMAP协议性能测试
- `stress_test.sh` - 压力测试脚本
- `benchmark.py` - 基准测试工具

## 使用方法

```bash
# 运行API测试
node api_test.js

# 运行SMTP测试
go run smtp_test.go

# 运行IMAP测试
go run imap_test.go

# 运行压力测试
./stress_test.sh

# 运行基准测试
python3 benchmark.py
```

## 测试结果

测试结果将保存在 `results/` 目录中，包含：
- 性能指标报告
- 响应时间统计
- 系统资源使用情况
- 错误日志分析