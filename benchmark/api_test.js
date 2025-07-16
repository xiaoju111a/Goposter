#!/usr/bin/env node

/**
 * API性能测试脚本
 * 测试邮箱服务器API接口的性能和稳定性
 */

const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');

class APITester {
    constructor(baseURL = 'http://localhost:9090') {
        this.baseURL = baseURL;
        this.results = {
            tests: [],
            summary: {
                totalRequests: 0,
                successfulRequests: 0,
                failedRequests: 0,
                averageResponseTime: 0,
                minResponseTime: Infinity,
                maxResponseTime: 0
            }
        };
    }

    // 发送HTTP请求
    async request(method, path, data = null, headers = {}) {
        const url = new URL(path, this.baseURL);
        const options = {
            hostname: url.hostname,
            port: url.port || (url.protocol === 'https:' ? 443 : 80),
            path: url.pathname,
            method: method.toUpperCase(),
            headers: {
                'Content-Type': 'application/json',
                ...headers
            }
        };

        if (data) {
            const jsonData = JSON.stringify(data);
            options.headers['Content-Length'] = Buffer.byteLength(jsonData);
        }

        const startTime = Date.now();
        
        return new Promise((resolve, reject) => {
            const req = (url.protocol === 'https:' ? https : http).request(options, (res) => {
                let responseData = '';
                
                res.on('data', (chunk) => {
                    responseData += chunk;
                });
                
                res.on('end', () => {
                    const endTime = Date.now();
                    const responseTime = endTime - startTime;
                    
                    try {
                        const parsed = responseData ? JSON.parse(responseData) : null;
                        resolve({
                            statusCode: res.statusCode,
                            data: parsed,
                            responseTime,
                            headers: res.headers
                        });
                    } catch (error) {
                        resolve({
                            statusCode: res.statusCode,
                            data: responseData,
                            responseTime,
                            headers: res.headers
                        });
                    }
                });
            });

            req.on('error', (error) => {
                const endTime = Date.now();
                const responseTime = endTime - startTime;
                reject({
                    error: error.message,
                    responseTime
                });
            });

            if (data) {
                req.write(JSON.stringify(data));
            }

            req.end();
        });
    }

    // 记录测试结果
    recordResult(testName, success, responseTime, error = null) {
        this.results.tests.push({
            testName,
            success,
            responseTime,
            error,
            timestamp: new Date().toISOString()
        });

        this.results.summary.totalRequests++;
        if (success) {
            this.results.summary.successfulRequests++;
        } else {
            this.results.summary.failedRequests++;
        }

        this.results.summary.minResponseTime = Math.min(this.results.summary.minResponseTime, responseTime);
        this.results.summary.maxResponseTime = Math.max(this.results.summary.maxResponseTime, responseTime);
    }

    // 测试邮箱列表API
    async testMailboxesAPI(iterations = 10) {
        console.log(`\\n📮 测试邮箱列表API (${iterations}次)...`);
        
        for (let i = 0; i < iterations; i++) {
            try {
                const response = await this.request('GET', '/api/mailboxes');
                const success = response.statusCode === 200;
                this.recordResult('GET /api/mailboxes', success, response.responseTime);
                
                if (success) {
                    console.log(`✅ 请求${i + 1}: ${response.responseTime}ms - 获取${response.data.length}个邮箱`);
                } else {
                    console.log(`❌ 请求${i + 1}: ${response.responseTime}ms - 状态码: ${response.statusCode}`);
                }
            } catch (error) {
                this.recordResult('GET /api/mailboxes', false, error.responseTime, error.error);
                console.log(`❌ 请求${i + 1}: ${error.responseTime}ms - 错误: ${error.error}`);
            }
            
            // 短暂延迟避免过度请求
            await new Promise(resolve => setTimeout(resolve, 100));
        }
    }

    // 测试邮件发送API
    async testSendEmailAPI(iterations = 5) {
        console.log(`\\n📤 测试邮件发送API (${iterations}次)...`);
        
        for (let i = 0; i < iterations; i++) {
            const testEmail = {
                from: 'test@ygocard.org',
                to: 'recipient@example.com',
                subject: `性能测试邮件 ${i + 1}`,
                body: `这是第${i + 1}封性能测试邮件\\n发送时间: ${new Date().toISOString()}`
            };

            try {
                const response = await this.request('POST', '/api/send', testEmail);
                const success = response.statusCode === 200;
                this.recordResult('POST /api/send', success, response.responseTime);
                
                if (success) {
                    console.log(`✅ 邮件${i + 1}: ${response.responseTime}ms - 发送成功`);
                } else {
                    console.log(`❌ 邮件${i + 1}: ${response.responseTime}ms - 状态码: ${response.statusCode}`);
                }
            } catch (error) {
                this.recordResult('POST /api/send', false, error.responseTime, error.error);
                console.log(`❌ 邮件${i + 1}: ${error.responseTime}ms - 错误: ${error.error}`);
            }
            
            // 邮件发送间隔稍长
            await new Promise(resolve => setTimeout(resolve, 500));
        }
    }

    // 测试认证API
    async testAuthAPI(iterations = 10) {
        console.log(`\\n🔐 测试认证API (${iterations}次)...`);
        
        for (let i = 0; i < iterations; i++) {
            const credentials = {
                email: 'admin@ygocard.org',
                password: 'admin123'
            };

            try {
                const response = await this.request('POST', '/api/auth/login', credentials);
                const success = response.statusCode === 200;
                this.recordResult('POST /api/auth/login', success, response.responseTime);
                
                if (success) {
                    console.log(`✅ 认证${i + 1}: ${response.responseTime}ms - 登录成功`);
                } else {
                    console.log(`❌ 认证${i + 1}: ${response.responseTime}ms - 状态码: ${response.statusCode}`);
                }
            } catch (error) {
                this.recordResult('POST /api/auth/login', false, error.responseTime, error.error);
                console.log(`❌ 认证${i + 1}: ${error.responseTime}ms - 错误: ${error.error}`);
            }
            
            await new Promise(resolve => setTimeout(resolve, 200));
        }
    }

    // 并发测试
    async testConcurrentRequests(concurrency = 10) {
        console.log(`\\n🚀 并发测试 (${concurrency}个并发请求)...`);
        
        const promises = [];
        const startTime = Date.now();
        
        for (let i = 0; i < concurrency; i++) {
            promises.push(this.request('GET', '/api/mailboxes'));
        }

        try {
            const responses = await Promise.allSettled(promises);
            const endTime = Date.now();
            const totalTime = endTime - startTime;
            
            let successful = 0;
            let failed = 0;
            
            responses.forEach((result, index) => {
                if (result.status === 'fulfilled') {
                    const success = result.value.statusCode === 200;
                    this.recordResult(`Concurrent GET /api/mailboxes #${index + 1}`, success, result.value.responseTime);
                    if (success) successful++;
                    else failed++;
                } else {
                    this.recordResult(`Concurrent GET /api/mailboxes #${index + 1}`, false, 0, result.reason);
                    failed++;
                }
            });
            
            console.log(`✅ 并发测试完成: ${successful}成功, ${failed}失败, 总耗时: ${totalTime}ms`);
        } catch (error) {
            console.log(`❌ 并发测试失败: ${error.message}`);
        }
    }

    // 计算统计信息
    calculateStats() {
        const responseTimes = this.results.tests.map(test => test.responseTime);
        const totalTime = responseTimes.reduce((sum, time) => sum + time, 0);
        
        this.results.summary.averageResponseTime = totalTime / responseTimes.length;
        
        // 计算成功率
        this.results.summary.successRate = (this.results.summary.successfulRequests / this.results.summary.totalRequests) * 100;
        
        // 计算吞吐量 (每秒请求数)
        const testDuration = this.results.tests.length > 0 ? 
            (new Date(this.results.tests[this.results.tests.length - 1].timestamp) - 
             new Date(this.results.tests[0].timestamp)) / 1000 : 0;
        
        this.results.summary.throughput = testDuration > 0 ? this.results.summary.totalRequests / testDuration : 0;
    }

    // 生成测试报告
    generateReport() {
        this.calculateStats();
        
        const report = {
            timestamp: new Date().toISOString(),
            testResults: this.results,
            systemInfo: {
                nodeVersion: process.version,
                platform: process.platform,
                arch: process.arch,
                memory: process.memoryUsage()
            }
        };
        
        // 保存详细报告
        const reportPath = path.join(__dirname, 'results', `api_test_${Date.now()}.json`);
        fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
        
        // 控制台输出摘要
        console.log('\\n📊 测试报告摘要:');
        console.log('=====================================');
        console.log(`总请求数: ${this.results.summary.totalRequests}`);
        console.log(`成功请求: ${this.results.summary.successfulRequests}`);
        console.log(`失败请求: ${this.results.summary.failedRequests}`);
        console.log(`成功率: ${this.results.summary.successRate.toFixed(2)}%`);
        console.log(`平均响应时间: ${this.results.summary.averageResponseTime.toFixed(2)}ms`);
        console.log(`最快响应时间: ${this.results.summary.minResponseTime}ms`);
        console.log(`最慢响应时间: ${this.results.summary.maxResponseTime}ms`);
        console.log(`吞吐量: ${this.results.summary.throughput.toFixed(2)} req/s`);
        console.log('=====================================');
        console.log(`详细报告已保存至: ${reportPath}`);
    }

    // 运行所有测试
    async runAllTests() {
        console.log('🧪 开始邮箱服务器API性能测试...');
        console.log(`目标服务器: ${this.baseURL}`);
        
        await this.testMailboxesAPI(10);
        await this.testAuthAPI(10);
        await this.testSendEmailAPI(5);
        await this.testConcurrentRequests(10);
        
        this.generateReport();
        
        console.log('\\n✅ 所有测试完成！');
    }
}

// 主函数
async function main() {
    const tester = new APITester();
    await tester.runAllTests();
}

// 运行测试
if (require.main === module) {
    main().catch(console.error);
}

module.exports = APITester;