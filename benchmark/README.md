# goposter 邮箱服务器性能测试

本目录包含 goposter 邮箱服务器的综合性能测试工具。

## 测试功能

### 🔍 综合性能测试
- **Web API 性能** - 邮箱列表、认证、邮件发送等API响应时间测试
- **SMTP 协议测试** - SMTP连接性能和邮件发送速度测试
- **IMAP 协议测试** - IMAP连接性能和邮件读取速度测试
- **并发性能测试** - 高并发场景下的系统稳定性测试
- **系统资源监控** - 内存、CPU使用情况监控

## 测试工具

### 核心测试脚本
- `comprehensive_test.py` - 综合性能测试工具，包含所有测试功能

### 使用方法

```bash
# 运行综合性能测试
cd benchmark
python3 comprehensive_test.py

# 测试特定组件
python3 comprehensive_test.py --web-only    # 仅测试Web API
python3 comprehensive_test.py --smtp-only   # 仅测试SMTP
python3 comprehensive_test.py --imap-only   # 仅测试IMAP
```

## 测试结果

测试结果保存在 `results/` 目录中：

- **`PERFORMANCE_SUMMARY.json`** - 性能测试汇总数据
- **`PERFORMANCE_TEST_FINAL_REPORT.md`** - 详细性能测试报告
- **`performance_report_*.md`** - 历史测试报告

## 性能指标

测试涵盖以下关键性能指标：
- API响应时间和吞吐量
- SMTP/IMAP连接性能
- 并发处理能力
- 系统资源使用效率
- 错误率和稳定性

## 环境要求

- Python 3.7+
- 邮箱服务器运行在本地 (localhost:9090)
- SMTP端口: 25, IMAP端口: 143