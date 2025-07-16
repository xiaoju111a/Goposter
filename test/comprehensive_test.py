#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
邮箱服务器综合性能测试工具
基于当前ygocard.org架构的完整性能测试
"""

import urllib.request
import urllib.parse
import json
import time
import threading
import statistics
import os
import socket
import smtplib
from datetime import datetime
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

class ComprehensiveTest:
    def __init__(self, 
                 web_url="http://localhost:9090",
                 smtp_host="localhost",
                 smtp_port=25,
                 imap_host="localhost", 
                 imap_port=143,
                 domain="ygocard.org"):
        self.web_url = web_url
        self.smtp_host = smtp_host
        self.smtp_port = smtp_port
        self.imap_host = imap_host
        self.imap_port = imap_port
        self.domain = domain
        self.results = []
        self.start_time = time.time()
        
    def log(self, message):
        timestamp = datetime.now().strftime("%H:%M:%S")
        print(f"[{timestamp}] {message}")
        
    def http_request(self, url, method="GET", data=None, headers=None):
        """发送HTTP请求"""
        start_time = time.time()
        
        try:
            if data:
                data = json.dumps(data).encode('utf-8')
                req = urllib.request.Request(url, data=data, method=method)
                req.add_header('Content-Type', 'application/json')
            else:
                req = urllib.request.Request(url, method=method)
            
            if headers:
                for key, value in headers.items():
                    req.add_header(key, value)
            
            with urllib.request.urlopen(req, timeout=15) as response:
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
    
    def test_web_api_performance(self):
        """测试Web API性能"""
        self.log("🌐 测试Web API性能...")
        
        api_tests = [
            ('GET', '/api/mailboxes', None, '获取邮箱列表'),
            ('GET', '/api/stats', None, '获取统计信息'),
            ('GET', '/api/config', None, '获取系统配置'),
            ('GET', '/api/dns/config', None, '获取DNS配置'),
            ('GET', '/api/relay/status', None, '获取中继状态'),
        ]
        
        test_results = []
        
        for method, endpoint, data, description in api_tests:
            self.log(f"  测试 {method} {endpoint} - {description}")
            
            response_times = []
            success_count = 0
            
            for i in range(10):
                url = f"{self.web_url}{endpoint}"
                result = self.http_request(url, method, data)
                response_times.append(result['response_time'])
                
                if result['success'] and result['status_code'] == 200:
                    success_count += 1
                    if i < 3:  # 只打印前3次的详细信息
                        print(f"    ✅ 请求{i+1}: {result['response_time']:.2f}ms")
                else:
                    if i < 3:
                        error = result.get('error', f"HTTP {result['status_code']}")
                        print(f"    ❌ 请求{i+1}: {result['response_time']:.2f}ms - {error}")
                
                time.sleep(0.1)
            
            if response_times:
                avg_time = statistics.mean(response_times)
                min_time = min(response_times)
                max_time = max(response_times)
                success_rate = (success_count / 10) * 100
                
                test_results.append({
                    'endpoint': endpoint,
                    'method': method,
                    'description': description,
                    'requests': 10,
                    'success_count': success_count,
                    'success_rate': success_rate,
                    'avg_response_time': avg_time,
                    'min_response_time': min_time,
                    'max_response_time': max_time
                })
                
                print(f"    📊 {description}: 成功率 {success_rate:.1f}%, 平均响应时间 {avg_time:.2f}ms")
        
        return {
            'test_name': 'Web API Performance',
            'results': test_results
        }
    
    def test_authentication_performance(self):
        """测试认证性能"""
        self.log("🔐 测试认证性能...")
        
        auth_data = {
            "email": f"admin@{self.domain}",
            "password": "admin123"
        }
        
        response_times = []
        success_count = 0
        
        for i in range(15):
            url = f"{self.web_url}/api/auth/login"
            result = self.http_request(url, "POST", auth_data)
            response_times.append(result['response_time'])
            
            if result['success'] and result['status_code'] == 200:
                success_count += 1
                if i < 3:
                    print(f"  ✅ 认证{i+1}: {result['response_time']:.2f}ms - 登录成功")
            else:
                if i < 3:
                    error = result.get('error', f"HTTP {result['status_code']}")
                    print(f"  ❌ 认证{i+1}: {result['response_time']:.2f}ms - {error}")
            
            time.sleep(0.2)
        
        if response_times:
            avg_time = statistics.mean(response_times)
            success_rate = (success_count / 15) * 100
            
            print(f"  📊 认证性能: 成功率 {success_rate:.1f}%, 平均响应时间 {avg_time:.2f}ms")
            
            return {
                'test_name': 'Authentication Performance',
                'requests': 15,
                'success_count': success_count,
                'success_rate': success_rate,
                'avg_response_time': avg_time
            }
        
        return None
    
    def test_smtp_performance(self):
        """测试SMTP性能"""
        self.log("📧 测试SMTP性能...")
        
        smtp_results = []
        
        # 测试SMTP连接
        connection_times = []
        connection_success = 0
        
        for i in range(10):
            start_time = time.time()
            try:
                sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                sock.settimeout(10)
                sock.connect((self.smtp_host, self.smtp_port))
                sock.close()
                
                end_time = time.time()
                response_time = (end_time - start_time) * 1000
                connection_times.append(response_time)
                connection_success += 1
                
                if i < 3:
                    print(f"  ✅ SMTP连接{i+1}: {response_time:.2f}ms")
                    
            except Exception as e:
                end_time = time.time()
                response_time = (end_time - start_time) * 1000
                connection_times.append(response_time)
                
                if i < 3:
                    print(f"  ❌ SMTP连接{i+1}: {response_time:.2f}ms - {str(e)}")
            
            time.sleep(0.1)
        
        if connection_times:
            avg_conn_time = statistics.mean(connection_times)
            conn_success_rate = (connection_success / 10) * 100
            
            smtp_results.append({
                'test_type': 'SMTP Connection',
                'attempts': 10,
                'success_count': connection_success,
                'success_rate': conn_success_rate,
                'avg_response_time': avg_conn_time
            })
            
            print(f"  📊 SMTP连接: 成功率 {conn_success_rate:.1f}%, 平均连接时间 {avg_conn_time:.2f}ms")
        
        return {
            'test_name': 'SMTP Performance',
            'results': smtp_results
        }
    
    def test_concurrent_performance(self):
        """测试并发性能"""
        self.log("🚀 测试并发性能...")
        
        concurrent_levels = [5, 10, 20, 50]
        concurrent_results = []
        
        for concurrency in concurrent_levels:
            self.log(f"  测试 {concurrency} 并发用户...")
            
            requests_per_user = 5
            total_requests = concurrency * requests_per_user
            
            results = []
            start_time = time.time()
            
            def worker():
                for _ in range(requests_per_user):
                    url = f"{self.web_url}/api/mailboxes"
                    result = self.http_request(url)
                    results.append(result)
                    time.sleep(0.05)
            
            # 创建并启动线程
            threads = []
            for _ in range(concurrency):
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
            response_times = [r['response_time'] for r in results if r['success']]
            
            if response_times:
                avg_time = statistics.mean(response_times)
                min_time = min(response_times)
                max_time = max(response_times)
                throughput = total_requests / total_time
                success_rate = (success_count / total_requests) * 100
                
                concurrent_results.append({
                    'concurrency': concurrency,
                    'total_requests': total_requests,
                    'success_count': success_count,
                    'success_rate': success_rate,
                    'total_time': total_time,
                    'avg_response_time': avg_time,
                    'min_response_time': min_time,
                    'max_response_time': max_time,
                    'throughput': throughput
                })
                
                print(f"    📊 {concurrency}并发: 成功率 {success_rate:.1f}%, "
                      f"吞吐量 {throughput:.2f} req/s, 平均响应时间 {avg_time:.2f}ms")
        
        return {
            'test_name': 'Concurrent Performance',
            'results': concurrent_results
        }
    
    def test_system_endpoints(self):
        """测试系统端点"""
        self.log("⚙️  测试系统端点...")
        
        system_endpoints = [
            ('/api/admin/users', 'GET', '管理员用户列表'),
            ('/api/admin/mailboxes', 'GET', '管理员邮箱列表'),
            ('/api/forwarding/settings', 'GET', '转发设置'),
            ('/api/relay/providers', 'GET', '中继提供商'),
        ]
        
        endpoint_results = []
        
        for endpoint, method, description in system_endpoints:
            response_times = []
            success_count = 0
            
            for i in range(5):
                url = f"{self.web_url}{endpoint}"
                result = self.http_request(url, method)
                response_times.append(result['response_time'])
                
                if result['success']:
                    success_count += 1
            
            if response_times:
                avg_time = statistics.mean(response_times)
                success_rate = (success_count / 5) * 100
                
                endpoint_results.append({
                    'endpoint': endpoint,
                    'description': description,
                    'success_rate': success_rate,
                    'avg_response_time': avg_time
                })
                
                print(f"  📊 {description}: 成功率 {success_rate:.1f}%, 平均响应时间 {avg_time:.2f}ms")
        
        return {
            'test_name': 'System Endpoints',
            'results': endpoint_results
        }
    
    def run_comprehensive_test(self):
        """运行综合性能测试"""
        self.log("🧪 开始综合性能测试")
        self.log(f"目标服务器: {self.web_url}")
        self.log(f"SMTP服务器: {self.smtp_host}:{self.smtp_port}")
        self.log(f"IMAP服务器: {self.imap_host}:{self.imap_port}")
        self.log(f"测试域名: {self.domain}")
        
        # 执行各项测试
        test_functions = [
            self.test_web_api_performance,
            self.test_authentication_performance,
            self.test_smtp_performance,
            self.test_concurrent_performance,
            self.test_system_endpoints
        ]
        
        for test_func in test_functions:
            print("\\n" + "="*60)
            try:
                result = test_func()
                if result:
                    self.results.append(result)
                time.sleep(1)  # 测试间隔
            except Exception as e:
                self.log(f"❌ 测试失败: {test_func.__name__} - {str(e)}")
        
        self.generate_comprehensive_report()
    
    def generate_comprehensive_report(self):
        """生成综合测试报告"""
        self.log("📋 生成综合测试报告")
        
        end_time = time.time()
        total_test_time = end_time - self.start_time
        
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        report_file = f"results/comprehensive_test_{timestamp}.json"
        
        try:
            os.makedirs("results", exist_ok=True)
            
            # 生成报告数据
            report_data = {
                'timestamp': datetime.now().isoformat(),
                'test_duration': total_test_time,
                'target_servers': {
                    'web': self.web_url,
                    'smtp': f"{self.smtp_host}:{self.smtp_port}",
                    'imap': f"{self.imap_host}:{self.imap_port}",
                    'domain': self.domain
                },
                'test_results': self.results,
                'system_info': {
                    'python_version': f"{__import__('sys').version}",
                    'platform': f"{__import__('platform').system()} {__import__('platform').release()}",
                    'test_time': datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                }
            }
            
            # 保存详细报告
            with open(report_file, 'w', encoding='utf-8') as f:
                json.dump(report_data, f, indent=2, ensure_ascii=False)
            
            self.log(f"详细报告已保存: {report_file}")
            
        except Exception as e:
            self.log(f"保存报告失败: {e}")
        
        # 生成控制台摘要
        self.print_summary_report(total_test_time)
        
        # 生成Markdown报告
        self.generate_markdown_report(report_data, timestamp)
    
    def print_summary_report(self, total_test_time):
        """打印摘要报告"""
        print("\\n" + "="*80)
        print("📊 YgoCard邮箱服务器综合性能测试报告")
        print("="*80)
        print(f"测试时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        print(f"总测试时长: {total_test_time:.2f}秒")
        print(f"目标服务器: {self.web_url}")
        print(f"测试域名: {self.domain}")
        print("-"*80)
        
        # 统计各项测试结果
        total_tests = 0
        total_success = 0
        
        for result in self.results:
            print(f"\\n{result['test_name']}:")
            
            if 'results' in result:
                for sub_result in result['results']:
                    if 'success_count' in sub_result and 'requests' in sub_result:
                        total_tests += sub_result['requests']
                        total_success += sub_result['success_count']
                        print(f"  - {sub_result.get('description', sub_result.get('endpoint', 'N/A'))}: "
                              f"成功率 {sub_result['success_rate']:.1f}%")
            elif 'success_count' in result:
                total_tests += result.get('requests', 0)
                total_success += result['success_count']
                print(f"  - 成功率: {result['success_rate']:.1f}%")
        
        print("-"*80)
        if total_tests > 0:
            overall_success_rate = (total_success / total_tests) * 100
            print(f"总体测试成功率: {overall_success_rate:.1f}% ({total_success}/{total_tests})")
        
        print("="*80)
        print("✅ 综合性能测试完成！")
    
    def generate_markdown_report(self, report_data, timestamp):
        """生成Markdown格式报告"""
        try:
            md_file = f"results/performance_report_{timestamp}.md"
            
            with open(md_file, 'w', encoding='utf-8') as f:
                f.write("# YgoCard邮箱服务器性能测试报告\\n\\n")
                f.write(f"**测试时间:** {report_data['timestamp']}\\n")
                f.write(f"**测试时长:** {report_data['test_duration']:.2f}秒\\n")
                f.write(f"**目标服务器:** {report_data['target_servers']['web']}\\n")
                f.write(f"**测试域名:** {report_data['target_servers']['domain']}\\n\\n")
                
                f.write("## 测试结果概览\\n\\n")
                
                for result in report_data['test_results']:
                    f.write(f"### {result['test_name']}\\n\\n")
                    
                    if 'results' in result:
                        f.write("| 测试项 | 成功率 | 平均响应时间 | 说明 |\\n")
                        f.write("|--------|--------|--------------|------|\\n")
                        
                        for sub_result in result['results']:
                            name = sub_result.get('description', sub_result.get('endpoint', 'N/A'))
                            success_rate = sub_result.get('success_rate', 0)
                            avg_time = sub_result.get('avg_response_time', 0)
                            
                            f.write(f"| {name} | {success_rate:.1f}% | {avg_time:.2f}ms | - |\\n")
                    
                    elif 'success_rate' in result:
                        f.write(f"- **成功率:** {result['success_rate']:.1f}%\\n")
                        f.write(f"- **平均响应时间:** {result.get('avg_response_time', 0):.2f}ms\\n")
                    
                    f.write("\\n")
                
                f.write("## 系统信息\\n\\n")
                f.write(f"- **Python版本:** {report_data['system_info']['python_version']}\\n")
                f.write(f"- **操作系统:** {report_data['system_info']['platform']}\\n")
                f.write(f"- **测试工具:** YgoCard邮箱服务器综合性能测试工具\\n")
            
            self.log(f"Markdown报告已保存: {md_file}")
            
        except Exception as e:
            self.log(f"生成Markdown报告失败: {e}")

def main():
    """主函数"""
    os.makedirs("results", exist_ok=True)
    
    # 创建测试实例 - 适配当前架构
    tester = ComprehensiveTest(
        web_url="http://localhost:9090",
        smtp_host="localhost",
        smtp_port=25,
        imap_host="localhost",
        imap_port=143,
        domain="ygocard.org"
    )
    
    try:
        tester.run_comprehensive_test()
    except KeyboardInterrupt:
        print("\\n⚠️ 测试被用户中断")
        tester.generate_comprehensive_report()
    except Exception as e:
        print(f"\\n❌ 测试过程中发生错误: {e}")
        tester.generate_comprehensive_report()

if __name__ == "__main__":
    main()