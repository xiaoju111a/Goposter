#!/usr/bin/env node
/**
 * 高并发邮箱服务器性能测试工具
 * 测试50-100并发场景下的服务器性能表现
 */

const http = require('http');
const https = require('https');
const { URL } = require('url');
const fs = require('fs');
const path = require('path');

class HighConcurrencyTester {
    constructor(baseUrl = 'http://localhost:9090') {
        this.baseUrl = baseUrl;
        this.results = [];
        this.activeConnections = 0;
        this.maxActiveConnections = 0;
        this.startTime = null;
        this.endTime = null;
    }

    log(message) {
        const timestamp = new Date().toTimeString().split(' ')[0];
        console.log(`[${timestamp}] ${message}`);
    }

    async makeRequest(method, endpoint, data = null, timeout = 10000) {
        return new Promise((resolve) => {
            const url = new URL(endpoint, this.baseUrl);
            const options = {
                hostname: url.hostname,
                port: url.port,
                path: url.pathname,
                method: method,
                timeout: timeout,
                headers: {
                    'Content-Type': 'application/json',
                    'User-Agent': 'HighConcurrencyTester/1.0'
                }
            };

            if (data) {
                const postData = JSON.stringify(data);
                options.headers['Content-Length'] = Buffer.byteLength(postData);
            }

            const startTime = Date.now();
            this.activeConnections++;
            this.maxActiveConnections = Math.max(this.maxActiveConnections, this.activeConnections);

            const client = url.protocol === 'https:' ? https : http;
            const req = client.request(options, (res) => {
                let responseData = '';
                
                res.on('data', (chunk) => {
                    responseData += chunk;
                });

                res.on('end', () => {
                    const endTime = Date.now();
                    this.activeConnections--;
                    
                    resolve({
                        success: res.statusCode >= 200 && res.statusCode < 300,
                        statusCode: res.statusCode,
                        responseTime: endTime - startTime,
                        data: responseData,
                        error: null
                    });
                });
            });

            req.on('error', (err) => {
                const endTime = Date.now();
                this.activeConnections--;
                
                resolve({
                    success: false,
                    statusCode: 0,
                    responseTime: endTime - startTime,
                    data: null,
                    error: err.message
                });
            });

            req.on('timeout', () => {
                req.destroy();
                const endTime = Date.now();
                this.activeConnections--;
                
                resolve({
                    success: false,
                    statusCode: 0,
                    responseTime: endTime - startTime,
                    data: null,
                    error: 'Request timeout'
                });
            });

            if (data) {
                req.write(JSON.stringify(data));
            }
            
            req.end();
        });
    }

    async testConcurrentAPI(concurrency, requestsPerWorker = 10) {
        this.log(`🚀 开始${concurrency}并发API测试 (每个worker${requestsPerWorker}个请求)...`);
        
        const workers = [];
        const results = [];
        this.startTime = Date.now();

        // 创建并发workers
        for (let i = 0; i < concurrency; i++) {
            const worker = this.runWorker(i + 1, requestsPerWorker, results);
            workers.push(worker);
        }

        // 等待所有workers完成
        await Promise.all(workers);
        this.endTime = Date.now();

        // 统计结果
        const totalRequests = results.length;
        const successfulRequests = results.filter(r => r.success).length;
        const failedRequests = totalRequests - successfulRequests;
        const responseTimes = results.map(r => r.responseTime);
        const avgResponseTime = responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length;
        const minResponseTime = Math.min(...responseTimes);
        const maxResponseTime = Math.max(...responseTimes);
        const totalTime = this.endTime - this.startTime;
        const throughput = totalRequests / (totalTime / 1000);

        const testResult = {
            test: `${concurrency} Concurrent API Test`,
            concurrency: concurrency,
            requestsPerWorker: requestsPerWorker,
            totalRequests: totalRequests,
            successfulRequests: successfulRequests,
            failedRequests: failedRequests,
            successRate: (successfulRequests / totalRequests) * 100,
            avgResponseTime: avgResponseTime,
            minResponseTime: minResponseTime,
            maxResponseTime: maxResponseTime,
            totalTime: totalTime,
            throughput: throughput,
            maxActiveConnections: this.maxActiveConnections,
            timestamp: new Date().toISOString()
        };

        this.results.push(testResult);
        this.log(`✅ ${concurrency}并发测试完成: 成功率${testResult.successRate.toFixed(1)}%, 吞吐量${throughput.toFixed(2)} req/s`);
        
        return testResult;
    }

    async runWorker(workerId, requestCount, results) {
        for (let i = 0; i < requestCount; i++) {
            const result = await this.makeRequest('GET', '/api/mailboxes');
            result.workerId = workerId;
            result.requestIndex = i + 1;
            result.timestamp = Date.now();
            results.push(result);

            // 小延迟避免过度集中
            await new Promise(resolve => setTimeout(resolve, Math.random() * 50));
        }
    }

    async testConcurrentSMTP(concurrency, emailsPerWorker = 3) {
        this.log(`📧 开始${concurrency}并发SMTP测试 (每个worker${emailsPerWorker}封邮件)...`);
        
        const workers = [];
        const results = [];
        this.startTime = Date.now();

        // 创建并发workers
        for (let i = 0; i < concurrency; i++) {
            const worker = this.runSMTPWorker(i + 1, emailsPerWorker, results);
            workers.push(worker);
        }

        // 等待所有workers完成
        await Promise.all(workers);
        this.endTime = Date.now();

        // 统计结果
        const totalEmails = results.length;
        const successfulEmails = results.filter(r => r.success).length;
        const failedEmails = totalEmails - successfulEmails;
        const responseTimes = results.map(r => r.responseTime);
        const avgResponseTime = responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length;
        const totalTime = this.endTime - this.startTime;
        const throughput = totalEmails / (totalTime / 1000);

        const testResult = {
            test: `${concurrency} Concurrent SMTP Test`,
            concurrency: concurrency,
            emailsPerWorker: emailsPerWorker,
            totalEmails: totalEmails,
            successfulEmails: successfulEmails,
            failedEmails: failedEmails,
            successRate: (successfulEmails / totalEmails) * 100,
            avgResponseTime: avgResponseTime,
            totalTime: totalTime,
            throughput: throughput,
            timestamp: new Date().toISOString()
        };

        this.results.push(testResult);
        this.log(`✅ ${concurrency}并发SMTP测试完成: 成功率${testResult.successRate.toFixed(1)}%, 吞吐量${throughput.toFixed(2)} emails/s`);
        
        return testResult;
    }

    async runSMTPWorker(workerId, emailCount, results) {
        for (let i = 0; i < emailCount; i++) {
            const emailData = {
                from: 'test@freeagent.live',
                to: 'recipient@example.com',
                subject: `高并发测试邮件 Worker${workerId}-${i + 1}`,
                body: `这是来自Worker ${workerId}的第${i + 1}封高并发测试邮件\n发送时间: ${new Date().toISOString()}`
            };

            const result = await this.makeRequest('POST', '/api/send', emailData);
            result.workerId = workerId;
            result.emailIndex = i + 1;
            result.timestamp = Date.now();
            results.push(result);

            // 邮件发送间隔
            await new Promise(resolve => setTimeout(resolve, 100 + Math.random() * 200));
        }
    }

    async testLoadStability(concurrency, duration = 60) {
        this.log(`⏱️ 开始${concurrency}并发负载稳定性测试 (持续${duration}秒)...`);
        
        const results = [];
        const workers = [];
        this.startTime = Date.now();
        const endTime = this.startTime + (duration * 1000);

        // 创建持续运行的workers
        for (let i = 0; i < concurrency; i++) {
            const worker = this.runStabilityWorker(i + 1, endTime, results);
            workers.push(worker);
        }

        // 等待所有workers完成
        await Promise.all(workers);
        this.endTime = Date.now();

        // 统计结果
        const totalRequests = results.length;
        const successfulRequests = results.filter(r => r.success).length;
        const actualDuration = (this.endTime - this.startTime) / 1000;
        const throughput = totalRequests / actualDuration;

        const testResult = {
            test: `${concurrency} Concurrent Stability Test`,
            concurrency: concurrency,
            duration: actualDuration,
            totalRequests: totalRequests,
            successfulRequests: successfulRequests,
            successRate: (successfulRequests / totalRequests) * 100,
            throughput: throughput,
            timestamp: new Date().toISOString()
        };

        this.results.push(testResult);
        this.log(`✅ ${concurrency}并发稳定性测试完成: 持续${actualDuration.toFixed(1)}s, 吞吐量${throughput.toFixed(2)} req/s`);
        
        return testResult;
    }

    async runStabilityWorker(workerId, endTime, results) {
        let requestCount = 0;
        
        while (Date.now() < endTime) {
            requestCount++;
            const result = await this.makeRequest('GET', '/api/mailboxes');
            result.workerId = workerId;
            result.requestIndex = requestCount;
            result.timestamp = Date.now();
            results.push(result);

            // 动态调整请求间隔
            const interval = 200 + Math.random() * 300;
            await new Promise(resolve => setTimeout(resolve, interval));
        }
    }

    async runHighConcurrencyBenchmark() {
        this.log('🧪 开始高并发基准测试套件...');
        this.log(`目标服务器: ${this.baseUrl}`);
        
        const concurrencyLevels = [50, 75, 100];
        
        console.log('\n' + '='.repeat(60));
        console.log('📊 高并发API性能测试');
        console.log('='.repeat(60));
        
        for (const concurrency of concurrencyLevels) {
            await this.testConcurrentAPI(concurrency, 5);
            await new Promise(resolve => setTimeout(resolve, 2000)); // 间隔2秒
        }

        console.log('\n' + '='.repeat(60));
        console.log('📧 高并发SMTP性能测试');
        console.log('='.repeat(60));
        
        const smtpConcurrencyLevels = [20, 30, 50]; // SMTP并发相对保守
        for (const concurrency of smtpConcurrencyLevels) {
            await this.testConcurrentSMTP(concurrency, 2);
            await new Promise(resolve => setTimeout(resolve, 3000)); // 间隔3秒
        }

        console.log('\n' + '='.repeat(60));
        console.log('⏱️ 高并发稳定性测试');
        console.log('='.repeat(60));
        
        await this.testLoadStability(30, 30); // 30并发持续30秒

        this.generateReport();
    }

    generateReport() {
        this.log('📋 生成高并发测试报告...');
        
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
        const reportFile = `results/high_concurrency_report_${timestamp}.json`;
        
        try {
            // 确保results目录存在
            if (!fs.existsSync('results')) {
                fs.mkdirSync('results', { recursive: true });
            }

            // 生成详细报告
            const report = {
                timestamp: new Date().toISOString(),
                testSuite: 'High Concurrency Benchmark',
                targetServer: this.baseUrl,
                systemInfo: {
                    nodeVersion: process.version,
                    platform: process.platform,
                    arch: process.arch,
                    memory: process.memoryUsage()
                },
                results: this.results,
                summary: this.generateSummary()
            };

            fs.writeFileSync(reportFile, JSON.stringify(report, null, 2));
            this.log(`✅ 详细报告已保存: ${reportFile}`);
        } catch (error) {
            this.log(`❌ 保存报告失败: ${error.message}`);
        }

        // 控制台摘要
        console.log('\n' + '='.repeat(80));
        console.log('📊 高并发测试报告摘要');
        console.log('='.repeat(80));
        
        this.results.forEach(result => {
            console.log(`\n${result.test}:`);
            Object.entries(result).forEach(([key, value]) => {
                if (key !== 'test' && key !== 'timestamp') {
                    if (typeof value === 'number') {
                        console.log(`  ${key}: ${value.toFixed(2)}`);
                    } else {
                        console.log(`  ${key}: ${value}`);
                    }
                }
            });
        });

        console.log('\n' + '='.repeat(80));
        console.log('🎯 性能建议:');
        console.log('='.repeat(80));
        this.generateRecommendations();
        
        console.log('\n✅ 高并发测试完成！');
    }

    generateSummary() {
        const apiTests = this.results.filter(r => r.test.includes('API'));
        const smtpTests = this.results.filter(r => r.test.includes('SMTP'));
        const stabilityTests = this.results.filter(r => r.test.includes('Stability'));

        return {
            totalTests: this.results.length,
            apiTests: {
                count: apiTests.length,
                avgSuccessRate: apiTests.reduce((sum, test) => sum + test.successRate, 0) / apiTests.length,
                maxThroughput: Math.max(...apiTests.map(test => test.throughput))
            },
            smtpTests: {
                count: smtpTests.length,
                avgSuccessRate: smtpTests.reduce((sum, test) => sum + test.successRate, 0) / smtpTests.length,
                maxThroughput: Math.max(...smtpTests.map(test => test.throughput))
            },
            stabilityTests: {
                count: stabilityTests.length,
                avgSuccessRate: stabilityTests.reduce((sum, test) => sum + test.successRate, 0) / stabilityTests.length
            }
        };
    }

    generateRecommendations() {
        const apiTests = this.results.filter(r => r.test.includes('API'));
        const maxConcurrency = Math.max(...apiTests.map(t => t.concurrency));
        const maxSuccessRate = Math.max(...apiTests.map(t => t.successRate));
        const maxThroughput = Math.max(...apiTests.map(t => t.throughput));

        console.log(`• 最高并发处理能力: ${maxConcurrency}个并发连接`);
        console.log(`• 最佳成功率: ${maxSuccessRate.toFixed(1)}%`);
        console.log(`• 峰值吞吐量: ${maxThroughput.toFixed(2)} req/s`);
        
        if (maxSuccessRate < 95) {
            console.log(`• ⚠️ 建议优化: 成功率低于95%，考虑增加连接池或优化数据库查询`);
        }
        
        if (maxThroughput < 100) {
            console.log(`• ⚠️ 建议优化: 吞吐量较低，考虑启用HTTP Keep-Alive或增加Worker进程`);
        }
        
        console.log(`• 💡 建议: 在生产环境中建议使用负载均衡器处理超过${maxConcurrency}的并发`);
    }
}

async function main() {
    const tester = new HighConcurrencyTester();
    
    try {
        await tester.runHighConcurrencyBenchmark();
    } catch (error) {
        console.error(`\n❌ 测试过程中发生错误: ${error.message}`);
        tester.generateReport();
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}

module.exports = HighConcurrencyTester;