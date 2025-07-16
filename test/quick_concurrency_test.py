#!/usr/bin/env python3
"""
快速高并发测试工具
针对50-100并发场景的快速评估
"""

import asyncio
import aiohttp
import time
import statistics
import json
from datetime import datetime
import concurrent.futures
import threading
import signal
import sys

class QuickConcurrencyTester:
    def __init__(self, base_url="http://localhost:9090"):
        self.base_url = base_url
        self.results = []
        self.total_requests = 0
        self.successful_requests = 0
        self.failed_requests = 0
        self.running = True
        
    def log(self, message):
        timestamp = datetime.now().strftime("%H:%M:%S")
        print(f"[{timestamp}] {message}")
        
    async def make_request(self, session, endpoint="/api/mailboxes"):
        """异步HTTP请求"""
        start_time = time.time()
        try:
            async with session.get(f"{self.base_url}{endpoint}", timeout=10) as response:
                end_time = time.time()
                response_time = (end_time - start_time) * 1000
                
                return {
                    'success': 200 <= response.status < 300,
                    'status_code': response.status,
                    'response_time': response_time,
                    'timestamp': end_time
                }
        except Exception as e:
            end_time = time.time()
            response_time = (end_time - start_time) * 1000
            
            return {
                'success': False,
                'status_code': 0,
                'response_time': response_time,
                'error': str(e),
                'timestamp': end_time
            }
    
    async def concurrent_api_test(self, concurrency, requests_per_worker=10):
        """并发API测试"""
        self.log(f"🚀 开始{concurrency}并发API测试 (每worker {requests_per_worker}请求)")
        
        start_time = time.time()
        results = []
        
        # 创建aiohttp session
        connector = aiohttp.TCPConnector(limit=concurrency*2, limit_per_host=concurrency*2)
        timeout = aiohttp.ClientTimeout(total=30)
        
        async with aiohttp.ClientSession(connector=connector, timeout=timeout) as session:
            # 创建所有任务
            tasks = []
            for worker_id in range(concurrency):
                for req_id in range(requests_per_worker):
                    task = self.make_request(session)
                    tasks.append(task)
            
            # 执行所有请求
            results = await asyncio.gather(*tasks, return_exceptions=True)
        
        end_time = time.time()
        total_time = end_time - start_time
        
        # 处理结果
        valid_results = [r for r in results if isinstance(r, dict)]
        successful = sum(1 for r in valid_results if r['success'])
        failed = len(valid_results) - successful
        
        if valid_results:
            response_times = [r['response_time'] for r in valid_results]
            avg_response_time = statistics.mean(response_times)
            min_response_time = min(response_times)
            max_response_time = max(response_times)
        else:
            avg_response_time = min_response_time = max_response_time = 0
        
        throughput = len(valid_results) / total_time
        success_rate = (successful / len(valid_results)) * 100 if valid_results else 0
        
        result = {
            'test': f'{concurrency} Concurrent API Test',
            'concurrency': concurrency,
            'total_requests': len(valid_results),
            'successful_requests': successful,
            'failed_requests': failed,
            'success_rate': success_rate,
            'avg_response_time': avg_response_time,
            'min_response_time': min_response_time,
            'max_response_time': max_response_time,
            'total_time': total_time,
            'throughput': throughput,
            'timestamp': datetime.now().isoformat()
        }
        
        self.results.append(result)
        self.log(f"✅ {concurrency}并发测试完成: 成功率{success_rate:.1f}%, 吞吐量{throughput:.2f} req/s")
        
        return result
    
    def sync_smtp_test(self, concurrency, emails_per_worker=3):
        """同步SMTP测试（使用线程池）"""
        self.log(f"📧 开始{concurrency}并发SMTP测试 (每worker {emails_per_worker}邮件)")
        
        import requests
        
        def send_email(worker_id, email_id):
            try:
                start_time = time.time()
                
                email_data = {
                    "from": "test@ygocard.org",
                    "to": "recipient@example.com", 
                    "subject": f"快速并发测试 Worker{worker_id}-{email_id}",
                    "body": f"这是Worker {worker_id}的第{email_id}封测试邮件\n时间: {datetime.now().isoformat()}"
                }
                
                response = requests.post(
                    f"{self.base_url}/api/send",
                    json=email_data,
                    timeout=10
                )
                
                end_time = time.time()
                response_time = (end_time - start_time) * 1000
                
                return {
                    'success': 200 <= response.status_code < 300,
                    'status_code': response.status_code,
                    'response_time': response_time,
                    'worker_id': worker_id,
                    'email_id': email_id
                }
                
            except Exception as e:
                end_time = time.time()
                response_time = (end_time - start_time) * 1000
                
                return {
                    'success': False,
                    'status_code': 0,
                    'response_time': response_time,
                    'error': str(e),
                    'worker_id': worker_id,
                    'email_id': email_id
                }
        
        start_time = time.time()
        
        # 使用线程池执行SMTP测试
        with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as executor:
            futures = []
            
            for worker_id in range(concurrency):
                for email_id in range(emails_per_worker):
                    future = executor.submit(send_email, worker_id, email_id)
                    futures.append(future)
            
            # 收集结果
            results = []
            for future in concurrent.futures.as_completed(futures):
                try:
                    result = future.result(timeout=15)
                    results.append(result)
                except Exception as e:
                    results.append({
                        'success': False,
                        'error': str(e),
                        'response_time': 15000
                    })
        
        end_time = time.time()
        total_time = end_time - start_time
        
        # 统计结果
        successful = sum(1 for r in results if r['success'])
        failed = len(results) - successful
        
        if results:
            response_times = [r['response_time'] for r in results]
            avg_response_time = statistics.mean(response_times)
        else:
            avg_response_time = 0
        
        throughput = len(results) / total_time
        success_rate = (successful / len(results)) * 100 if results else 0
        
        result = {
            'test': f'{concurrency} Concurrent SMTP Test',
            'concurrency': concurrency,
            'total_emails': len(results),
            'successful_emails': successful,
            'failed_emails': failed,
            'success_rate': success_rate,
            'avg_response_time': avg_response_time,
            'total_time': total_time,
            'throughput': throughput,
            'timestamp': datetime.now().isoformat()
        }
        
        self.results.append(result)
        self.log(f"✅ {concurrency}并发SMTP测试完成: 成功率{success_rate:.1f}%, 吞吐量{throughput:.2f} emails/s")
        
        return result
    
    async def run_quick_benchmark(self):
        """运行快速基准测试"""
        self.log("🧪 开始快速高并发基准测试")
        self.log(f"目标服务器: {self.base_url}")
        
        # API并发测试
        print("\n" + "="*60)
        print("📊 高并发API性能测试")
        print("="*60)
        
        api_concurrency_levels = [50, 75, 100]
        for concurrency in api_concurrency_levels:
            await self.concurrent_api_test(concurrency, 5)
            await asyncio.sleep(1)  # 短暂休息
        
        # SMTP并发测试
        print("\n" + "="*60)
        print("📧 高并发SMTP性能测试")
        print("="*60)
        
        smtp_concurrency_levels = [20, 35, 50]
        for concurrency in smtp_concurrency_levels:
            self.sync_smtp_test(concurrency, 2)
            time.sleep(2)  # 休息2秒
        
        self.generate_report()
    
    def generate_report(self):
        """生成测试报告"""
        self.log("📋 生成快速测试报告")
        
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        report_file = f"results/quick_concurrency_report_{timestamp}.json"
        
        try:
            report = {
                'timestamp': datetime.now().isoformat(),
                'test_suite': 'Quick High Concurrency Test',
                'target_server': self.base_url,
                'results': self.results,
                'summary': self.generate_summary()
            }
            
            with open(report_file, 'w', encoding='utf-8') as f:
                json.dump(report, f, indent=2, ensure_ascii=False)
            
            self.log(f"✅ 详细报告已保存: {report_file}")
        except Exception as e:
            self.log(f"❌ 保存报告失败: {e}")
        
        # 控制台摘要
        print("\n" + "="*80)
        print("📊 快速高并发测试摘要")
        print("="*80)
        
        for result in self.results:
            print(f"\n{result['test']}:")
            key_metrics = ['concurrency', 'success_rate', 'avg_response_time', 'throughput']
            for key in key_metrics:
                if key in result:
                    value = result[key]
                    if isinstance(value, float):
                        print(f"  {key}: {value:.2f}")
                    else:
                        print(f"  {key}: {value}")
        
        print("\n" + "="*80)
        print("🎯 快速评估结论:")
        print("="*80)
        
        api_tests = [r for r in self.results if 'API' in r['test']]
        smtp_tests = [r for r in self.results if 'SMTP' in r['test']]
        
        if api_tests:
            max_api_concurrency = max(r['concurrency'] for r in api_tests)
            max_api_throughput = max(r['throughput'] for r in api_tests)
            avg_api_success_rate = statistics.mean(r['success_rate'] for r in api_tests)
            
            print(f"• API最高并发: {max_api_concurrency}个连接")
            print(f"• API峰值吞吐量: {max_api_throughput:.2f} req/s") 
            print(f"• API平均成功率: {avg_api_success_rate:.1f}%")
        
        if smtp_tests:
            max_smtp_concurrency = max(r['concurrency'] for r in smtp_tests)
            max_smtp_throughput = max(r['throughput'] for r in smtp_tests)
            avg_smtp_success_rate = statistics.mean(r['success_rate'] for r in smtp_tests)
            
            print(f"• SMTP最高并发: {max_smtp_concurrency}个连接")
            print(f"• SMTP峰值吞吐量: {max_smtp_throughput:.2f} emails/s")
            print(f"• SMTP平均成功率: {avg_smtp_success_rate:.1f}%")
        
        print("\n✅ 快速高并发测试完成！")
    
    def generate_summary(self):
        """生成摘要统计"""
        api_tests = [r for r in self.results if 'API' in r['test']]
        smtp_tests = [r for r in self.results if 'SMTP' in r['test']]
        
        summary = {
            'total_tests': len(self.results),
            'api_tests': len(api_tests),
            'smtp_tests': len(smtp_tests)
        }
        
        if api_tests:
            summary['max_api_concurrency'] = max(r['concurrency'] for r in api_tests)
            summary['max_api_throughput'] = max(r['throughput'] for r in api_tests)
            summary['avg_api_success_rate'] = statistics.mean(r['success_rate'] for r in api_tests)
        
        if smtp_tests:
            summary['max_smtp_concurrency'] = max(r['concurrency'] for r in smtp_tests)
            summary['max_smtp_throughput'] = max(r['throughput'] for r in smtp_tests)
            summary['avg_smtp_success_rate'] = statistics.mean(r['success_rate'] for r in smtp_tests)
        
        return summary

def signal_handler(sig, frame):
    print('\n⚠️ 测试被用户中断')
    sys.exit(0)

async def main():
    signal.signal(signal.SIGINT, signal_handler)
    
    tester = QuickConcurrencyTester()
    
    try:
        await tester.run_quick_benchmark()
    except KeyboardInterrupt:
        tester.log("⚠️ 测试被用户中断")
        tester.generate_report()
    except Exception as e:
        tester.log(f"❌ 测试过程中发生错误: {e}")
        tester.generate_report()

if __name__ == "__main__":
    import os
    os.makedirs("results", exist_ok=True)
    asyncio.run(main())