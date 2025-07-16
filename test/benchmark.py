#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
邮箱服务器基准测试工具
综合测试API性能、邮件处理能力和系统资源使用情况
"""

import requests
import time
import threading
import queue
import statistics
import json
import os
import sys
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor, as_completed
import subprocess
import psutil

class MailServerBenchmark:
    def __init__(self, base_url="http://localhost:9090"):
        self.base_url = base_url
        self.results = {
            "api_tests": [],
            "email_tests": [],
            "concurrent_tests": [],
            "resource_usage": [],
            "summary": {}
        }
        self.start_time = time.time()
        
    def log(self, message, level="INFO"):
        """日志输出"""
        timestamp = datetime.now().strftime("%H:%M:%S")
        print(f"[{timestamp}] {level}: {message}")
        
    def measure_response_time(self, func, *args, **kwargs):
        """测量函数执行时间"""
        start_time = time.time()
        try:
            result = func(*args, **kwargs)
            end_time = time.time()
            return (end_time - start_time) * 1000, True, result, None
        except Exception as e:
            end_time = time.time()
            return (end_time - start_time) * 1000, False, None, str(e)
    
    def test_api_endpoint(self, endpoint, method="GET", data=None, iterations=10):
        """测试API端点性能"""
        self.log(f"🔍 测试API端点: {method} {endpoint} ({iterations}次)")
        
        response_times = []
        success_count = 0
        error_count = 0
        errors = []
        
        for i in range(iterations):
            url = f"{self.base_url}{endpoint}"
            
            if method.upper() == "GET":
                response_time, success, result, error = self.measure_response_time(
                    requests.get, url, timeout=10
                )
            elif method.upper() == "POST":
                response_time, success, result, error = self.measure_response_time(
                    requests.post, url, json=data, timeout=10
                )
            else:
                continue
                
            response_times.append(response_time)
            
            if success and result.status_code == 200:
                success_count += 1
                print(f"✅ 请求{i+1}: {response_time:.2f}ms")
            else:
                error_count += 1
                error_msg = error or f"HTTP {result.status_code if result else 'Unknown'}"
                errors.append(error_msg)
                print(f"❌ 请求{i+1}: {response_time:.2f}ms - {error_msg}")
            
            time.sleep(0.1)  # 避免过度请求
        
        # 计算统计信息
        avg_time = statistics.mean(response_times)
        min_time = min(response_times)
        max_time = max(response_times)
        median_time = statistics.median(response_times)
        
        test_result = {
            "endpoint": endpoint,
            "method": method,
            "iterations": iterations,
            "success_count": success_count,
            "error_count": error_count,
            "success_rate": (success_count / iterations) * 100,
            "avg_response_time": avg_time,
            "min_response_time": min_time,
            "max_response_time": max_time,
            "median_response_time": median_time,
            "errors": errors[:5]  # 只保留前5个错误
        }
        
        self.results["api_tests"].append(test_result)
        
        self.log(f"📊 结果: 成功率 {test_result['success_rate']:.1f}%, "
                f"平均响应时间 {avg_time:.2f}ms")
        
        return test_result
    
    def test_concurrent_requests(self, endpoint, concurrent_users=10, requests_per_user=5):
        """测试并发请求性能"""
        self.log(f"🚀 并发测试: {concurrent_users}个用户, 每用户{requests_per_user}个请求")
        
        total_requests = concurrent_users * requests_per_user
        response_times = []
        success_count = 0
        error_count = 0
        start_time = time.time()
        
        def worker(user_id):
            """工作线程函数"""
            user_results = []
            for i in range(requests_per_user):
                url = f"{self.base_url}{endpoint}"
                response_time, success, result, error = self.measure_response_time(
                    requests.get, url, timeout=10
                )
                
                user_results.append({
                    "user_id": user_id,
                    "request_id": i + 1,
                    "response_time": response_time,
                    "success": success,
                    "error": error
                })
                
                time.sleep(0.05)  # 短暂延迟
            
            return user_results
        
        # 使用线程池执行并发请求
        with ThreadPoolExecutor(max_workers=concurrent_users) as executor:
            futures = [executor.submit(worker, i+1) for i in range(concurrent_users)]
            
            for future in as_completed(futures):
                user_results = future.result()
                for result in user_results:
                    response_times.append(result["response_time"])
                    if result["success"]:
                        success_count += 1
                    else:
                        error_count += 1
        
        end_time = time.time()
        total_time = end_time - start_time
        
        # 计算统计信息
        avg_time = statistics.mean(response_times)
        throughput = total_requests / total_time
        
        concurrent_result = {
            "endpoint": endpoint,
            "concurrent_users": concurrent_users,
            "requests_per_user": requests_per_user,
            "total_requests": total_requests,
            "total_time": total_time,
            "success_count": success_count,
            "error_count": error_count,
            "success_rate": (success_count / total_requests) * 100,
            "avg_response_time": avg_time,
            "throughput": throughput,
            "min_response_time": min(response_times),
            "max_response_time": max(response_times)
        }
        
        self.results["concurrent_tests"].append(concurrent_result)
        
        self.log(f"📊 并发结果: 成功率 {concurrent_result['success_rate']:.1f}%, "
                f"吞吐量 {throughput:.2f} req/s, 平均响应时间 {avg_time:.2f}ms")
        
        return concurrent_result
    
    def test_email_sending(self, iterations=5):
        """测试邮件发送性能"""
        self.log(f"📧 测试邮件发送性能 ({iterations}次)")
        
        response_times = []
        success_count = 0
        error_count = 0
        
        for i in range(iterations):
            email_data = {
                "from": "test@ygocard.org",
                "to": "recipient@example.com",
                "subject": f"基准测试邮件 {i+1}",
                "body": f"这是第{i+1}封基准测试邮件\\n发送时间: {datetime.now().isoformat()}"
            }
            
            response_time, success, result, error = self.measure_response_time(
                requests.post, f"{self.base_url}/api/send", json=email_data, timeout=30
            )
            
            response_times.append(response_time)
            
            if success and result.status_code == 200:
                success_count += 1
                print(f"✅ 邮件{i+1}: {response_time:.2f}ms - 发送成功")
            else:
                error_count += 1
                error_msg = error or f"HTTP {result.status_code if result else 'Unknown'}"
                print(f"❌ 邮件{i+1}: {response_time:.2f}ms - {error_msg}")
            
            time.sleep(1)  # 邮件发送间隔更长
        
        # 计算统计信息
        avg_time = statistics.mean(response_times) if response_times else 0
        
        email_result = {
            "iterations": iterations,
            "success_count": success_count,
            "error_count": error_count,
            "success_rate": (success_count / iterations) * 100 if iterations > 0 else 0,
            "avg_response_time": avg_time,
            "min_response_time": min(response_times) if response_times else 0,
            "max_response_time": max(response_times) if response_times else 0
        }
        
        self.results["email_tests"].append(email_result)
        
        self.log(f"📊 邮件发送结果: 成功率 {email_result['success_rate']:.1f}%, "
                f"平均响应时间 {avg_time:.2f}ms")
        
        return email_result
    
    def monitor_system_resources(self, duration=60):
        """监控系统资源使用情况"""
        self.log(f"📊 监控系统资源 ({duration}秒)")
        
        resource_data = []
        start_time = time.time()
        
        while time.time() - start_time < duration:
            try:
                # CPU使用率
                cpu_percent = psutil.cpu_percent(interval=1)
                
                # 内存使用情况
                memory = psutil.virtual_memory()
                
                # 磁盘使用情况
                disk = psutil.disk_usage('/')
                
                # 网络统计
                network = psutil.net_io_counters()
                
                resource_info = {
                    "timestamp": datetime.now().isoformat(),
                    "cpu_percent": cpu_percent,
                    "memory_percent": memory.percent,
                    "memory_used_gb": memory.used / (1024**3),
                    "memory_total_gb": memory.total / (1024**3),
                    "disk_percent": (disk.used / disk.total) * 100,
                    "disk_used_gb": disk.used / (1024**3),
                    "network_bytes_sent": network.bytes_sent,
                    "network_bytes_recv": network.bytes_recv
                }
                
                resource_data.append(resource_info)
                
                if len(resource_data) % 10 == 0:  # 每10秒输出一次
                    print(f"📊 资源监控: CPU {cpu_percent:.1f}%, "
                          f"内存 {memory.percent:.1f}%, "
                          f"磁盘 {resource_info['disk_percent']:.1f}%")
                
            except Exception as e:
                self.log(f"资源监控错误: {e}", "ERROR")
                break
        
        self.results["resource_usage"] = resource_data
        
        if resource_data:
            avg_cpu = statistics.mean([r["cpu_percent"] for r in resource_data])
            avg_memory = statistics.mean([r["memory_percent"] for r in resource_data])
            
            self.log(f"📊 资源使用摘要: 平均CPU {avg_cpu:.1f}%, 平均内存 {avg_memory:.1f}%")
        
        return resource_data
    
    def run_full_benchmark(self):
        """运行完整的基准测试"""
        self.log("🧪 开始邮箱服务器基准测试")
        self.log(f"目标服务器: {self.base_url}")
        
        # 1. 基础API测试
        print("\n" + "="*50)
        print("📡 基础API性能测试")
        print("="*50)
        
        self.test_api_endpoint("/api/mailboxes", "GET", iterations=20)
        
        # 测试认证API
        auth_data = {
            "email": "admin@ygocard.org",
            "password": "admin123"
        }
        self.test_api_endpoint("/api/auth/login", "POST", auth_data, iterations=10)
        
        # 2. 邮件发送测试
        print("\n" + "="*50)
        print("📧 邮件发送性能测试")
        print("="*50)
        
        self.test_email_sending(iterations=5)
        
        # 3. 并发测试
        print("\n" + "="*50)
        print("🚀 并发性能测试")
        print("="*50)
        
        self.test_concurrent_requests("/api/mailboxes", concurrent_users=10, requests_per_user=5)
        self.test_concurrent_requests("/api/mailboxes", concurrent_users=20, requests_per_user=3)
        
        # 4. 系统资源监控（在后台运行，时间较短）
        print("\n" + "="*50)
        print("📊 系统资源监控")
        print("="*50)
        
        # 在后台启动资源监控
        import threading
        monitor_thread = threading.Thread(target=self.monitor_system_resources, args=(30,))
        monitor_thread.start()
        
        # 同时进行一些API请求以产生负载
        self.test_concurrent_requests("/api/mailboxes", concurrent_users=15, requests_per_user=10)
        
        # 等待监控完成
        monitor_thread.join()
        
        # 5. 生成报告
        self.generate_report()
    
    def generate_report(self):
        """生成测试报告"""
        self.log("📋 生成基准测试报告")
        
        total_time = time.time() - self.start_time
        
        # 计算总体统计
        all_api_tests = self.results["api_tests"]
        all_concurrent_tests = self.results["concurrent_tests"]
        all_email_tests = self.results["email_tests"]
        
        summary = {
            "test_duration": total_time,
            "total_api_tests": len(all_api_tests),
            "total_concurrent_tests": len(all_concurrent_tests),
            "total_email_tests": len(all_email_tests),
            "avg_api_response_time": 0,
            "avg_api_success_rate": 0,
            "max_throughput": 0,
            "avg_email_response_time": 0
        }
        
        if all_api_tests:
            summary["avg_api_response_time"] = statistics.mean([t["avg_response_time"] for t in all_api_tests])
            summary["avg_api_success_rate"] = statistics.mean([t["success_rate"] for t in all_api_tests])
        
        if all_concurrent_tests:
            summary["max_throughput"] = max([t["throughput"] for t in all_concurrent_tests])
        
        if all_email_tests:
            summary["avg_email_response_time"] = statistics.mean([t["avg_response_time"] for t in all_email_tests])
        
        self.results["summary"] = summary
        
        # 保存详细报告
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        report_file = f"results/benchmark_report_{timestamp}.json"
        
        try:
            with open(report_file, 'w', encoding='utf-8') as f:
                json.dump(self.results, f, indent=2, ensure_ascii=False)
            
            self.log(f"详细报告已保存: {report_file}")
        except Exception as e:
            self.log(f"保存报告失败: {e}", "ERROR")
        
        # 控制台输出摘要
        print("\n" + "="*60)
        print("📊 基准测试报告摘要")
        print("="*60)
        print(f"测试总时长: {total_time:.2f}秒")
        print(f"API测试数量: {summary['total_api_tests']}")
        print(f"并发测试数量: {summary['total_concurrent_tests']}")
        print(f"邮件测试数量: {summary['total_email_tests']}")
        print(f"平均API响应时间: {summary['avg_api_response_time']:.2f}ms")
        print(f"平均API成功率: {summary['avg_api_success_rate']:.1f}%")
        print(f"最大吞吐量: {summary['max_throughput']:.2f} req/s")
        print(f"平均邮件发送时间: {summary['avg_email_response_time']:.2f}ms")
        
        if self.results["resource_usage"]:
            resource_data = self.results["resource_usage"]
            avg_cpu = statistics.mean([r["cpu_percent"] for r in resource_data])
            avg_memory = statistics.mean([r["memory_percent"] for r in resource_data])
            print(f"平均CPU使用率: {avg_cpu:.1f}%")
            print(f"平均内存使用率: {avg_memory:.1f}%")
        
        print("="*60)
        print("✅ 基准测试完成！")

def main():
    """主函数"""
    # 检查依赖
    try:
        import requests
        import psutil
    except ImportError as e:
        print(f"❌ 缺少依赖包: {e}")
        print("请安装: pip install requests psutil")
        sys.exit(1)
    
    # 创建结果目录
    os.makedirs("results", exist_ok=True)
    
    # 运行基准测试
    benchmark = MailServerBenchmark()
    
    try:
        benchmark.run_full_benchmark()
    except KeyboardInterrupt:
        print("\n⚠️ 测试被用户中断")
        benchmark.generate_report()
    except Exception as e:
        print(f"\n❌ 测试过程中发生错误: {e}")
        benchmark.generate_report()

if __name__ == "__main__":
    main()