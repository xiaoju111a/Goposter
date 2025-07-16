#!/usr/bin/env python3
"""
简化版高并发测试工具 - 仅使用标准库
针对50-100并发场景的快速评估
"""

import urllib.request
import urllib.parse
import json
import time
import threading
import statistics
import queue
from datetime import datetime
import concurrent.futures
import socket

class SimpleConcurrencyTester:
    def __init__(self, base_url="http://localhost:9090"):
        self.base_url = base_url
        self.results = []
        self.lock = threading.Lock()
        
    def log(self, message):
        timestamp = datetime.now().strftime("%H:%M:%S")
        print(f"[{timestamp}] {message}")
        
    def http_request(self, endpoint="/api/mailboxes", method="GET", data=None, timeout=10):
        """发送HTTP请求"""
        start_time = time.time()
        
        try:
            url = f"{self.base_url}{endpoint}"
            
            if data:
                data = json.dumps(data).encode('utf-8')
                req = urllib.request.Request(url, data=data, method=method)
                req.add_header('Content-Type', 'application/json')
            else:
                req = urllib.request.Request(url, method=method)
            
            with urllib.request.urlopen(req, timeout=timeout) as response:
                response_data = response.read().decode('utf-8')
                status_code = response.getcode()
                
            end_time = time.time()
            response_time = (end_time - start_time) * 1000
            
            return {
                'success': 200 <= status_code < 300,
                'status_code': status_code,
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
    
    def smtp_connection_test(self, host="localhost", port=25, timeout=5):
        """SMTP连接测试"""
        start_time = time.time()
        
        try:
            # 创建TCP连接
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(timeout)
            sock.connect((host, port))
            
            # 读取欢迎消息
            welcome = sock.recv(1024).decode('utf-8')
            
            # 发送EHLO命令
            sock.send(b'EHLO test-client\r\n')
            ehlo_response = sock.recv(1024).decode('utf-8')
            
            # 发送QUIT命令
            sock.send(b'QUIT\r\n')
            quit_response = sock.recv(1024).decode('utf-8')
            
            sock.close()
            
            end_time = time.time()
            response_time = (end_time - start_time) * 1000
            
            return {
                'success': '250' in ehlo_response,
                'response_time': response_time,
                'timestamp': end_time
            }
            
        except Exception as e:
            end_time = time.time()
            response_time = (end_time - start_time) * 1000
            
            return {
                'success': False,
                'response_time': response_time,
                'error': str(e),
                'timestamp': end_time
            }
    
    def concurrent_api_test(self, concurrency, requests_per_worker=10):
        """并发API测试"""
        self.log(f"🚀 开始{concurrency}并发API测试 (每worker {requests_per_worker}请求)")
        
        start_time = time.time()
        results = []
        
        def worker(worker_id):
            worker_results = []
            for i in range(requests_per_worker):
                result = self.http_request()
                result['worker_id'] = worker_id
                result['request_id'] = i + 1
                worker_results.append(result)
                
                # 小延迟避免过度集中
                time.sleep(0.01 + (i % 10) * 0.001)
            
            with self.lock:
                results.extend(worker_results)
        
        # 使用线程池执行并发测试
        with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as executor:
            futures = [executor.submit(worker, i) for i in range(concurrency)]
            concurrent.futures.wait(futures)
        
        end_time = time.time()
        total_time = end_time - start_time
        
        # 统计结果
        successful = sum(1 for r in results if r['success'])
        failed = len(results) - successful
        
        if results:
            response_times = [r['response_time'] for r in results]
            avg_response_time = statistics.mean(response_times)
            min_response_time = min(response_times)
            max_response_time = max(response_times)
        else:
            avg_response_time = min_response_time = max_response_time = 0
        
        throughput = len(results) / total_time
        success_rate = (successful / len(results)) * 100 if results else 0
        
        result = {
            'test': f'{concurrency} Concurrent API Test',
            'concurrency': concurrency,
            'total_requests': len(results),
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
    
    def concurrent_smtp_test(self, concurrency, connections_per_worker=5):
        """并发SMTP连接测试"""
        self.log(f"📧 开始{concurrency}并发SMTP测试 (每worker {connections_per_worker}连接)")
        
        start_time = time.time()
        results = []
        
        def worker(worker_id):
            worker_results = []
            for i in range(connections_per_worker):
                result = self.smtp_connection_test()
                result['worker_id'] = worker_id
                result['connection_id'] = i + 1
                worker_results.append(result)
                
                # 连接间隔
                time.sleep(0.1 + (i % 5) * 0.02)
            
            with self.lock:
                results.extend(worker_results)
        
        # 使用线程池执行并发测试
        with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as executor:
            futures = [executor.submit(worker, i) for i in range(concurrency)]
            concurrent.futures.wait(futures)
        
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
            'total_connections': len(results),
            'successful_connections': successful,
            'failed_connections': failed,
            'success_rate': success_rate,
            'avg_response_time': avg_response_time,
            'total_time': total_time,
            'throughput': throughput,
            'timestamp': datetime.now().isoformat()
        }
        
        self.results.append(result)
        self.log(f"✅ {concurrency}并发SMTP测试完成: 成功率{success_rate:.1f}%, 吞吐量{throughput:.2f} conn/s")
        
        return result
    
    def burst_test(self, burst_size=100):
        """突发连接测试"""
        self.log(f"💥 开始突发连接测试 ({burst_size}个连接)")
        
        start_time = time.time()
        results = []
        
        def burst_worker(connection_id):
            result = self.smtp_connection_test()
            result['connection_id'] = connection_id
            return result
        
        # 同时发起大量连接
        with concurrent.futures.ThreadPoolExecutor(max_workers=burst_size) as executor:
            futures = [executor.submit(burst_worker, i) for i in range(burst_size)]
            results = [f.result() for f in concurrent.futures.as_completed(futures)]
        
        end_time = time.time()
        total_time = end_time - start_time
        
        successful = sum(1 for r in results if r['success'])
        success_rate = (successful / len(results)) * 100 if results else 0
        
        result = {
            'test': f'Burst Connection Test ({burst_size})',
            'burst_size': burst_size,
            'total_connections': len(results),
            'successful_connections': successful,
            'success_rate': success_rate,
            'total_time': total_time,
            'timestamp': datetime.now().isoformat()
        }
        
        self.results.append(result)
        self.log(f"✅ 突发测试完成: {successful}/{burst_size}成功 ({success_rate:.1f}%), 耗时{total_time:.2f}s")
        
        return result
    
    def run_benchmark(self):
        """运行完整基准测试"""
        self.log("🧪 开始简化版高并发基准测试")
        self.log(f"目标服务器: {self.base_url}")
        
        print("\n" + "="*60)
        print("📊 高并发API性能测试")
        print("="*60)
        
        # API并发测试
        api_levels = [50, 75, 100]
        for concurrency in api_levels:
            self.concurrent_api_test(concurrency, 8)
            time.sleep(1)
        
        print("\n" + "="*60)
        print("📧 高并发SMTP性能测试")
        print("="*60)
        
        # SMTP并发测试
        smtp_levels = [25, 40, 60]
        for concurrency in smtp_levels:
            self.concurrent_smtp_test(concurrency, 4)
            time.sleep(2)
        
        print("\n" + "="*60)
        print("💥 突发连接测试")
        print("="*60)
        
        # 突发测试
        burst_sizes = [50, 100, 150]
        for burst_size in burst_sizes:
            self.burst_test(burst_size)
            time.sleep(3)
        
        self.generate_report()
    
    def generate_report(self):
        """生成测试报告"""
        self.log("📋 生成测试报告")
        
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        report_file = f"results/simple_concurrency_report_{timestamp}.json"
        
        try:
            import os
            os.makedirs("results", exist_ok=True)
            
            report = {
                'timestamp': datetime.now().isoformat(),
                'test_suite': 'Simple High Concurrency Test',
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
        print("📊 高并发测试结果摘要")
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
        print("🎯 性能评估结论:")
        print("="*80)
        
        # 分析结果
        api_tests = [r for r in self.results if 'API' in r['test']]
        smtp_tests = [r for r in self.results if 'SMTP' in r['test']]
        burst_tests = [r for r in self.results if 'Burst' in r['test']]
        
        if api_tests:
            max_api_concurrency = max(r['concurrency'] for r in api_tests)
            max_api_throughput = max(r['throughput'] for r in api_tests) 
            avg_api_success_rate = statistics.mean(r['success_rate'] for r in api_tests)
            
            print(f"• API最高并发处理: {max_api_concurrency}个连接")
            print(f"• API峰值吞吐量: {max_api_throughput:.2f} req/s")
            print(f"• API平均成功率: {avg_api_success_rate:.1f}%")
        
        if smtp_tests:
            max_smtp_concurrency = max(r['concurrency'] for r in smtp_tests)
            max_smtp_throughput = max(r['throughput'] for r in smtp_tests)
            avg_smtp_success_rate = statistics.mean(r['success_rate'] for r in smtp_tests)
            
            print(f"• SMTP最高并发处理: {max_smtp_concurrency}个连接")
            print(f"• SMTP峰值吞吐量: {max_smtp_throughput:.2f} conn/s")
            print(f"• SMTP平均成功率: {avg_smtp_success_rate:.1f}%")
        
        if burst_tests:
            max_burst_size = max(r['burst_size'] for r in burst_tests)
            avg_burst_success_rate = statistics.mean(r['success_rate'] for r in burst_tests)
            
            print(f"• 最大突发处理: {max_burst_size}个连接")
            print(f"• 突发测试平均成功率: {avg_burst_success_rate:.1f}%")
        
        print("\n✅ 高并发测试完成！")
    
    def generate_summary(self):
        """生成摘要统计"""
        api_tests = [r for r in self.results if 'API' in r['test']]
        smtp_tests = [r for r in self.results if 'SMTP' in r['test']]
        burst_tests = [r for r in self.results if 'Burst' in r['test']]
        
        summary = {
            'total_tests': len(self.results),
            'api_tests': len(api_tests),
            'smtp_tests': len(smtp_tests),
            'burst_tests': len(burst_tests)
        }
        
        if api_tests:
            summary['max_api_concurrency'] = max(r['concurrency'] for r in api_tests)
            summary['max_api_throughput'] = max(r['throughput'] for r in api_tests)
            summary['avg_api_success_rate'] = statistics.mean(r['success_rate'] for r in api_tests)
        
        if smtp_tests:
            summary['max_smtp_concurrency'] = max(r['concurrency'] for r in smtp_tests)
            summary['max_smtp_throughput'] = max(r['throughput'] for r in smtp_tests)
            summary['avg_smtp_success_rate'] = statistics.mean(r['success_rate'] for r in smtp_tests)
        
        if burst_tests:
            summary['max_burst_size'] = max(r['burst_size'] for r in burst_tests)
            summary['avg_burst_success_rate'] = statistics.mean(r['success_rate'] for r in burst_tests)
        
        return summary

def main():
    tester = SimpleConcurrencyTester()
    
    try:
        tester.run_benchmark()
    except KeyboardInterrupt:
        tester.log("⚠️ 测试被用户中断")
        tester.generate_report()
    except Exception as e:
        tester.log(f"❌ 测试过程中发生错误: {e}")
        tester.generate_report()

if __name__ == "__main__":
    main()