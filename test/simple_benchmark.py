#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
简化版邮箱服务器基准测试工具
不依赖外部库，使用标准库进行测试
"""

import urllib.request
import urllib.parse
import json
import time
import threading
import statistics
import os
from datetime import datetime

class SimpleBenchmark:
    def __init__(self, base_url="http://localhost:9090"):
        self.base_url = base_url
        self.results = []
        
    def log(self, message):
        timestamp = datetime.now().strftime("%H:%M:%S")
        print(f"[{timestamp}] {message}")
        
    def http_request(self, url, method="GET", data=None):
        """发送HTTP请求"""
        start_time = time.time()
        
        try:
            if data:
                data = json.dumps(data).encode('utf-8')
                req = urllib.request.Request(url, data=data, method=method)
                req.add_header('Content-Type', 'application/json')
            else:
                req = urllib.request.Request(url, method=method)
            
            with urllib.request.urlopen(req, timeout=10) as response:
                response_data = response.read().decode('utf-8')
                status_code = response.getcode()
                
            end_time = time.time()
            response_time = (end_time - start_time) * 1000
            
            return {
                'success': True,
                'status_code': status_code,
                'response_time': response_time,
                'data': response_data
            }
            
        except Exception as e:
            end_time = time.time()
            response_time = (end_time - start_time) * 1000
            
            return {
                'success': False,
                'status_code': 0,
                'response_time': response_time,
                'error': str(e)
            }
    
    def test_api_performance(self):
        """测试API性能"""
        self.log("🔍 测试API性能...")
        
        # 测试邮箱列表API
        url = f"{self.base_url}/api/mailboxes"
        response_times = []
        success_count = 0
        
        for i in range(20):
            result = self.http_request(url)
            response_times.append(result['response_time'])
            
            if result['success'] and result['status_code'] == 200:
                success_count += 1
                data = json.loads(result['data'])
                print(f"✅ 请求{i+1}: {result['response_time']:.2f}ms - 获取{len(data)}个邮箱")
            else:
                error = result.get('error', f"HTTP {result['status_code']}")
                print(f"❌ 请求{i+1}: {result['response_time']:.2f}ms - {error}")
            
            time.sleep(0.1)
        
        avg_time = statistics.mean(response_times)
        min_time = min(response_times)
        max_time = max(response_times)
        success_rate = (success_count / 20) * 100
        
        self.log(f"📊 API性能结果: 成功率 {success_rate:.1f}%, 平均响应时间 {avg_time:.2f}ms")
        
        return {
            'test': 'API Performance',
            'requests': 20,
            'success_count': success_count,
            'success_rate': success_rate,
            'avg_response_time': avg_time,
            'min_response_time': min_time,
            'max_response_time': max_time
        }
    
    def test_concurrent_requests(self):
        """测试并发请求"""
        self.log("🚀 测试并发请求...")
        
        concurrent_users = 20
        requests_per_user = 5
        total_requests = concurrent_users * requests_per_user
        
        results = []
        start_time = time.time()
        
        def worker():
            for _ in range(requests_per_user):
                url = f"{self.base_url}/api/mailboxes"
                result = self.http_request(url)
                results.append(result)
                time.sleep(0.05)
        
        # 创建并启动线程
        threads = []
        for _ in range(concurrent_users):
            t = threading.Thread(target=worker)
            threads.append(t)
            t.start()
        
        # 等待所有线程完成
        for t in threads:
            t.join()
        
        end_time = time.time()
        total_time = end_time - start_time
        
        # 统计结果
        success_count = sum(1 for r in results if r['success'])
        response_times = [r['response_time'] for r in results]
        avg_time = statistics.mean(response_times)
        throughput = total_requests / total_time
        
        self.log(f"📊 并发测试结果: 成功率 {(success_count/total_requests)*100:.1f}%, "
                f"吞吐量 {throughput:.2f} req/s")
        
        return {
            'test': 'Concurrent Requests',
            'total_requests': total_requests,
            'concurrent_users': concurrent_users,
            'success_count': success_count,
            'total_time': total_time,
            'avg_response_time': avg_time,
            'throughput': throughput
        }
    
    def test_email_sending(self):
        """测试邮件发送"""
        self.log("📧 测试邮件发送...")
        
        response_times = []
        success_count = 0
        
        for i in range(3):  # 减少邮件发送测试次数
            email_data = {
                "from": "test@ygocard.org",
                "to": "recipient@example.com",
                "subject": f"基准测试邮件 {i+1}",
                "body": f"这是第{i+1}封基准测试邮件\\n发送时间: {datetime.now().isoformat()}"
            }
            
            url = f"{self.base_url}/api/send"
            result = self.http_request(url, "POST", email_data)
            response_times.append(result['response_time'])
            
            if result['success'] and result['status_code'] == 200:
                success_count += 1
                print(f"✅ 邮件{i+1}: {result['response_time']:.2f}ms - 发送成功")
            else:
                error = result.get('error', f"HTTP {result['status_code']}")
                print(f"❌ 邮件{i+1}: {result['response_time']:.2f}ms - {error}")
            
            time.sleep(1)
        
        if response_times:
            avg_time = statistics.mean(response_times)
            self.log(f"📊 邮件发送结果: 成功率 {(success_count/3)*100:.1f}%, "
                    f"平均响应时间 {avg_time:.2f}ms")
        
        return {
            'test': 'Email Sending',
            'emails_sent': 3,
            'success_count': success_count,
            'avg_response_time': avg_time if response_times else 0
        }
    
    def run_benchmark(self):
        """运行完整基准测试"""
        self.log("🧪 开始简化版基准测试")
        self.log(f"目标服务器: {self.base_url}")
        
        print("\\n" + "="*50)
        print("📡 API性能测试")
        print("="*50)
        api_result = self.test_api_performance()
        self.results.append(api_result)
        
        print("\\n" + "="*50)
        print("🚀 并发性能测试")  
        print("="*50)
        concurrent_result = self.test_concurrent_requests()
        self.results.append(concurrent_result)
        
        print("\\n" + "="*50)
        print("📧 邮件发送测试")
        print("="*50)
        email_result = self.test_email_sending()
        self.results.append(email_result)
        
        self.generate_report()
    
    def generate_report(self):
        """生成测试报告"""
        self.log("📋 生成基准测试报告")
        
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        report_file = f"results/simple_benchmark_{timestamp}.json"
        
        try:
            # 创建结果目录
            os.makedirs("results", exist_ok=True)
            
            # 保存详细报告
            with open(report_file, 'w', encoding='utf-8') as f:
                json.dump({
                    'timestamp': datetime.now().isoformat(),
                    'target_server': self.base_url,
                    'results': self.results
                }, f, indent=2, ensure_ascii=False)
            
            self.log(f"详细报告已保存: {report_file}")
        except Exception as e:
            self.log(f"保存报告失败: {e}")
        
        # 控制台输出摘要
        print("\\n" + "="*60)
        print("📊 基准测试报告摘要")
        print("="*60)
        
        for result in self.results:
            print(f"\\n{result['test']}:")
            for key, value in result.items():
                if key != 'test':
                    if isinstance(value, float):
                        print(f"  {key}: {value:.2f}")
                    else:
                        print(f"  {key}: {value}")
        
        print("="*60)
        print("✅ 基准测试完成！")

def main():
    """主函数"""
    os.makedirs("results", exist_ok=True)
    
    benchmark = SimpleBenchmark()
    
    try:
        benchmark.run_benchmark()
    except KeyboardInterrupt:
        print("\\n⚠️ 测试被用户中断")
        benchmark.generate_report()
    except Exception as e:
        print(f"\\n❌ 测试过程中发生错误: {e}")
        benchmark.generate_report()

if __name__ == "__main__":
    main()