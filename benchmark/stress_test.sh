#!/bin/bash

# 邮箱服务器压力测试脚本
# 综合测试服务器在高负载下的性能表现

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置参数
SERVER_HOST="localhost"
API_PORT="9090"
SMTP_PORT="25"
IMAP_PORT="143"
FRONTEND_PORT="8080"

# 测试参数
API_CONCURRENT=50
SMTP_CONCURRENT=20
IMAP_CONCURRENT=20
TEST_DURATION=60
EMAIL_COUNT=100

# 创建结果目录
mkdir -p results
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_DIR="results/stress_test_${TIMESTAMP}"
mkdir -p "$RESULT_DIR"

echo -e "${BLUE}🧪 开始邮箱服务器压力测试...${NC}"
echo -e "${BLUE}测试时间: $(date)${NC}"
echo -e "${BLUE}目标服务器: ${SERVER_HOST}${NC}"
echo -e "${BLUE}结果目录: ${RESULT_DIR}${NC}"
echo "========================================"

# 检查服务器状态
check_server_status() {
    echo -e "${YELLOW}🔍 检查服务器状态...${NC}"
    
    # 检查API端口
    if ncat -z "$SERVER_HOST" "$API_PORT" 2>/dev/null; then
        echo -e "${GREEN}✅ API服务器 (${API_PORT}) 正在运行${NC}"
    else
        echo -e "${RED}❌ API服务器 (${API_PORT}) 未运行${NC}"
        return 1
    fi
    
    # 检查SMTP端口
    if ncat -z "$SERVER_HOST" "$SMTP_PORT" 2>/dev/null; then
        echo -e "${GREEN}✅ SMTP服务器 (${SMTP_PORT}) 正在运行${NC}"
    else
        echo -e "${RED}❌ SMTP服务器 (${SMTP_PORT}) 未运行${NC}"
        return 1
    fi
    
    # 检查IMAP端口
    if ncat -z "$SERVER_HOST" "$IMAP_PORT" 2>/dev/null; then
        echo -e "${GREEN}✅ IMAP服务器 (${IMAP_PORT}) 正在运行${NC}"
    else
        echo -e "${RED}❌ IMAP服务器 (${IMAP_PORT}) 未运行${NC}"
        return 1
    fi
    
    # 检查前端端口
    if ncat -z "$SERVER_HOST" "$FRONTEND_PORT" 2>/dev/null; then
        echo -e "${GREEN}✅ 前端服务器 (${FRONTEND_PORT}) 正在运行${NC}"
    else
        echo -e "${RED}❌ 前端服务器 (${FRONTEND_PORT}) 未运行${NC}"
        return 1
    fi
    
    echo ""
}

# 监控系统资源
monitor_resources() {
    echo -e "${YELLOW}📊 开始监控系统资源...${NC}"
    
    # 获取邮箱服务器进程ID
    MAIL_PID=$(pgrep -f "go.*run.*main.go" | head -1)
    if [ -z "$MAIL_PID" ]; then
        echo -e "${RED}❌ 无法找到邮箱服务器进程${NC}"
        return 1
    fi
    
    echo "邮箱服务器进程ID: $MAIL_PID"
    
    # 后台监控资源使用情况
    {
        echo "时间,CPU使用率(%),内存使用率(%),内存使用量(MB),进程数,连接数"
        while true; do
            TIMESTAMP=$(date '+%H:%M:%S')
            
            # CPU和内存使用率
            if command -v top >/dev/null 2>&1; then
                CPU_MEM=$(top -p "$MAIL_PID" -bn1 | grep "$MAIL_PID" | awk '{print $9","$10}')
                if [ -z "$CPU_MEM" ]; then
                    CPU_MEM="0,0"
                fi
            else
                CPU_MEM="0,0"
            fi
            
            # 内存使用量(MB)
            if [ -f "/proc/$MAIL_PID/status" ]; then
                MEM_KB=$(grep "VmRSS" "/proc/$MAIL_PID/status" | awk '{print $2}')
                MEM_MB=$((MEM_KB / 1024))
            else
                MEM_MB=0
            fi
            
            # 进程数
            PROC_COUNT=$(pgrep -f "go.*run.*main.go" | wc -l)
            
            # 连接数
            CONN_COUNT=$(netstat -an | grep -E ":($API_PORT|$SMTP_PORT|$IMAP_PORT)" | grep ESTABLISHED | wc -l)
            
            echo "$TIMESTAMP,$CPU_MEM,$MEM_MB,$PROC_COUNT,$CONN_COUNT"
            sleep 5
        done
    } > "$RESULT_DIR/resource_monitor.csv" &
    
    MONITOR_PID=$!
    echo "资源监控进程ID: $MONITOR_PID"
    echo ""
}

# API压力测试
api_stress_test() {
    echo -e "${YELLOW}🚀 API压力测试 (${API_CONCURRENT}个并发连接)...${NC}"
    
    # 创建临时脚本
    cat > /tmp/api_stress.sh << 'EOF'
#!/bin/bash
API_URL="http://localhost:9090/api/mailboxes"
REQUESTS=0
SUCCESSFUL=0
FAILED=0

for i in $(seq 1 20); do
    REQUESTS=$((REQUESTS + 1))
    START_TIME=$(date +%s%3N)
    
    if curl -s -o /dev/null -w "%{http_code}" "$API_URL" | grep -q "200"; then
        SUCCESSFUL=$((SUCCESSFUL + 1))
    else
        FAILED=$((FAILED + 1))
    fi
    
    END_TIME=$(date +%s%3N)
    RESPONSE_TIME=$((END_TIME - START_TIME))
    
    echo "$(date '+%H:%M:%S'),$RESPONSE_TIME,$SUCCESSFUL,$FAILED"
    sleep 0.1
done
EOF
    
    chmod +x /tmp/api_stress.sh
    
    # 启动并发API测试
    echo "时间,响应时间(ms),成功数,失败数" > "$RESULT_DIR/api_stress.csv"
    
    for i in $(seq 1 $API_CONCURRENT); do
        /tmp/api_stress.sh >> "$RESULT_DIR/api_stress.csv" &
    done
    
    echo "等待API压力测试完成..."
    wait
    
    # 统计结果
    TOTAL_REQUESTS=$(tail -n +2 "$RESULT_DIR/api_stress.csv" | wc -l)
    TOTAL_SUCCESS=$(tail -n +2 "$RESULT_DIR/api_stress.csv" | tail -1 | cut -d',' -f3)
    TOTAL_FAILED=$(tail -n +2 "$RESULT_DIR/api_stress.csv" | tail -1 | cut -d',' -f4)
    
    echo -e "${GREEN}✅ API压力测试完成${NC}"
    echo "总请求数: $TOTAL_REQUESTS"
    echo "成功请求: $TOTAL_SUCCESS"
    echo "失败请求: $TOTAL_FAILED"
    echo ""
    
    rm -f /tmp/api_stress.sh
}

# SMTP压力测试
smtp_stress_test() {
    echo -e "${YELLOW}📧 SMTP压力测试 (${SMTP_CONCURRENT}个并发连接)...${NC}"
    
    # 创建临时邮件发送脚本
    cat > /tmp/smtp_stress.py << 'EOF'
#!/usr/bin/env python3
import smtplib
import time
import sys
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

def send_test_email(index):
    try:
        start_time = time.time()
        
        # 创建邮件
        msg = MIMEMultipart()
        msg['From'] = 'test@ygocard.org'
        msg['To'] = 'recipient@example.com'
        msg['Subject'] = f'SMTP压力测试邮件 {index}'
        
        body = f'这是第{index}封SMTP压力测试邮件\n发送时间: {time.strftime("%Y-%m-%d %H:%M:%S")}'
        msg.attach(MIMEText(body, 'plain'))
        
        # 发送邮件
        server = smtplib.SMTP('localhost', 25)
        server.sendmail('test@ygocard.org', ['recipient@example.com'], msg.as_string())
        server.quit()
        
        end_time = time.time()
        response_time = int((end_time - start_time) * 1000)
        
        print(f"{time.strftime('%H:%M:%S')},{response_time},1,0")
        
    except Exception as e:
        end_time = time.time()
        response_time = int((end_time - start_time) * 1000)
        print(f"{time.strftime('%H:%M:%S')},{response_time},0,1")

if __name__ == "__main__":
    worker_id = int(sys.argv[1]) if len(sys.argv) > 1 else 1
    
    for i in range(5):  # 每个工作进程发送5封邮件
        send_test_email(worker_id * 100 + i)
        time.sleep(0.5)
EOF
    
    chmod +x /tmp/smtp_stress.py
    
    # 启动并发SMTP测试
    echo "时间,响应时间(ms),成功数,失败数" > "$RESULT_DIR/smtp_stress.csv"
    
    if command -v python3 >/dev/null 2>&1; then
        for i in $(seq 1 $SMTP_CONCURRENT); do
            python3 /tmp/smtp_stress.py $i >> "$RESULT_DIR/smtp_stress.csv" &
        done
        
        echo "等待SMTP压力测试完成..."
        wait
        
        echo -e "${GREEN}✅ SMTP压力测试完成${NC}"
    else
        echo -e "${RED}❌ 未找到Python3，跳过SMTP测试${NC}"
    fi
    
    echo ""
    rm -f /tmp/smtp_stress.py
}

# 并发连接测试
concurrent_connection_test() {
    echo -e "${YELLOW}🔗 并发连接测试...${NC}"
    
    # 测试API并发连接
    echo "测试API并发连接..."
    for i in $(seq 1 100); do
        curl -s "http://$SERVER_HOST:$API_PORT/api/mailboxes" > /dev/null &
    done
    
    wait
    echo -e "${GREEN}✅ API并发连接测试完成${NC}"
    
    # 测试SMTP并发连接
    echo "测试SMTP并发连接..."
    for i in $(seq 1 50); do
        (echo "EHLO localhost"; sleep 0.1; echo "QUIT") | ncat "$SERVER_HOST" "$SMTP_PORT" > /dev/null 2>&1 &
    done
    
    wait
    echo -e "${GREEN}✅ SMTP并发连接测试完成${NC}"
    
    echo ""
}

# 长时间稳定性测试
stability_test() {
    echo -e "${YELLOW}⏱️ 长时间稳定性测试 (${TEST_DURATION}秒)...${NC}"
    
    START_TIME=$(date +%s)
    END_TIME=$((START_TIME + TEST_DURATION))
    
    echo "时间,API响应时间(ms),API状态,SMTP状态,IMAP状态" > "$RESULT_DIR/stability_test.csv"
    
    while [ $(date +%s) -lt $END_TIME ]; do
        TIMESTAMP=$(date '+%H:%M:%S')
        
        # 测试API响应时间
        API_START=$(date +%s%3N)
        if curl -s -o /dev/null "http://$SERVER_HOST:$API_PORT/api/mailboxes"; then
            API_END=$(date +%s%3N)
            API_TIME=$((API_END - API_START))
            API_STATUS="OK"
        else
            API_TIME=0
            API_STATUS="FAIL"
        fi
        
        # 测试SMTP连接
        if (echo "EHLO localhost"; echo "QUIT") | ncat "$SERVER_HOST" "$SMTP_PORT" > /dev/null 2>&1; then
            SMTP_STATUS="OK"
        else
            SMTP_STATUS="FAIL"
        fi
        
        # 测试IMAP连接
        if ncat -z "$SERVER_HOST" "$IMAP_PORT" 2>/dev/null; then
            IMAP_STATUS="OK"
        else
            IMAP_STATUS="FAIL"
        fi
        
        echo "$TIMESTAMP,$API_TIME,$API_STATUS,$SMTP_STATUS,$IMAP_STATUS" >> "$RESULT_DIR/stability_test.csv"
        
        sleep 5
    done
    
    echo -e "${GREEN}✅ 长时间稳定性测试完成${NC}"
    echo ""
}

# 生成测试报告
generate_report() {
    echo -e "${YELLOW}📋 生成测试报告...${NC}"
    
    cat > "$RESULT_DIR/test_report.txt" << EOF
邮箱服务器压力测试报告
=======================

测试时间: $(date)
测试服务器: $SERVER_HOST
测试持续时间: ${TEST_DURATION}秒

测试配置:
- API并发数: $API_CONCURRENT
- SMTP并发数: $SMTP_CONCURRENT
- IMAP并发数: $IMAP_CONCURRENT
- 邮件发送数: $EMAIL_COUNT

测试结果文件:
- api_stress.csv: API压力测试结果
- smtp_stress.csv: SMTP压力测试结果
- stability_test.csv: 稳定性测试结果
- resource_monitor.csv: 资源监控数据

测试总结:
EOF
    
    # 分析API测试结果
    if [ -f "$RESULT_DIR/api_stress.csv" ]; then
        API_TOTAL=$(tail -n +2 "$RESULT_DIR/api_stress.csv" | wc -l)
        API_AVG_TIME=$(tail -n +2 "$RESULT_DIR/api_stress.csv" | cut -d',' -f2 | awk '{sum+=$1} END {print sum/NR}')
        
        echo "API测试: $API_TOTAL 个请求, 平均响应时间: ${API_AVG_TIME}ms" >> "$RESULT_DIR/test_report.txt"
    fi
    
    # 分析稳定性测试结果
    if [ -f "$RESULT_DIR/stability_test.csv" ]; then
        STABILITY_TOTAL=$(tail -n +2 "$RESULT_DIR/stability_test.csv" | wc -l)
        STABILITY_OK=$(tail -n +2 "$RESULT_DIR/stability_test.csv" | cut -d',' -f3 | grep -c "OK")
        STABILITY_RATE=$(echo "scale=2; $STABILITY_OK * 100 / $STABILITY_TOTAL" | bc)
        
        echo "稳定性测试: $STABILITY_TOTAL 次检查, 成功率: ${STABILITY_RATE}%" >> "$RESULT_DIR/test_report.txt"
    fi
    
    echo -e "${GREEN}✅ 测试报告已生成: $RESULT_DIR/test_report.txt${NC}"
}

# 清理函数
cleanup() {
    echo -e "${YELLOW}🧹 清理测试环境...${NC}"
    
    # 停止资源监控
    if [ ! -z "$MONITOR_PID" ]; then
        kill $MONITOR_PID 2>/dev/null || true
    fi
    
    # 杀死所有后台进程
    jobs -p | xargs -r kill 2>/dev/null || true
    
    echo -e "${GREEN}✅ 清理完成${NC}"
}

# 主函数
main() {
    # 设置信号处理
    trap cleanup EXIT INT TERM
    
    # 检查依赖
    for cmd in curl ncat; do
        if ! command -v $cmd >/dev/null 2>&1; then
            echo -e "${RED}❌ 未找到命令: $cmd${NC}"
            exit 1
        fi
    done
    
    # 运行测试
    check_server_status
    monitor_resources
    
    api_stress_test
    smtp_stress_test
    concurrent_connection_test
    stability_test
    
    generate_report
    
    echo -e "${GREEN}🎉 压力测试完成！${NC}"
    echo -e "${GREEN}结果保存在: $RESULT_DIR${NC}"
}

# 运行主函数
main "$@"