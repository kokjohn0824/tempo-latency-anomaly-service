#!/bin/bash

# Tempo Latency Anomaly Service - Task Execution Script
# 用於快速執行各個任務的 Codex 指令

set -e

PROJECT_DIR="/Users/alexchang/dev/tempo-latency-anomaly-service"

echo "🚀 Tempo Latency Anomaly Service - Task Runner"
echo "============================================="
echo ""

# 顏色定義
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 任務列表
declare -A TASKS
TASKS[1]="Task 1: 初始化 Go 專案結構,建立所有目錄和空檔案佔位符"
TASKS[2]="Task 2: 實作 config 模組,支援 YAML 和環境變數"
TASKS[3]="Task 3: 實作 Redis 儲存層所有操作"
TASKS[4]="Task 4: 實作 domain 模型和 key 生成邏輯"
TASKS[5]="Task 5: 實作統計計算模組 (p50/p95/MAD)"
TASKS[6]="Task 6: 實作 Tempo HTTP 客戶端和查詢邏輯"
TASKS[7]="Task 7: 實作 service 層業務邏輯"
TASKS[8]="Task 8: 實作 HTTP API handlers 和路由"
TASKS[9]="Task 9: 實作背景任務輪詢和重算邏輯"
TASKS[10]="Task 10: 實作應用程式層 wiring 和 lifecycle"
TASKS[11]="Task 11: 建立 Dockerfile 和 docker-compose 配置"
TASKS[12]="Task 12: 完成 README 和測試資料"

# 函數: 顯示所有任務
show_tasks() {
    echo "可用任務列表:"
    echo ""
    for i in {1..12}; do
        echo -e "${BLUE}[$i]${NC} ${TASKS[$i]}"
    done
    echo ""
}

# 函數: 執行特定任務
run_task() {
    local task_num=$1
    
    if [[ -z "${TASKS[$task_num]}" ]]; then
        echo -e "${YELLOW}❌ 錯誤: 任務 $task_num 不存在${NC}"
        exit 1
    fi
    
    local task_desc="${TASKS[$task_num]}"
    
    echo -e "${GREEN}▶️  執行: $task_desc${NC}"
    echo ""
    
    cd "$PROJECT_DIR"
    
    # 執行 Codex
    export TERM=xterm
    codex exec "$task_desc" --full-auto
    
    echo ""
    echo -e "${GREEN}✅ 任務 $task_num 執行完成${NC}"
}

# 函數: 繼續下一個任務 (通用)
continue_next() {
    echo -e "${GREEN}▶️  繼續執行下一個任務...${NC}"
    echo ""
    
    cd "$PROJECT_DIR"
    export TERM=xterm
    codex exec "continue to next task" --full-auto
    
    echo ""
    echo -e "${GREEN}✅ 任務執行完成${NC}"
}

# 函數: 執行所有任務 (依序)
run_all() {
    echo -e "${YELLOW}⚠️  將依序執行所有 12 個任務${NC}"
    echo -e "${YELLOW}⚠️  這可能需要較長時間${NC}"
    echo ""
    read -p "確定要繼續嗎? (y/N) " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "已取消"
        exit 0
    fi
    
    for i in {1..12}; do
        echo ""
        echo -e "${BLUE}=====================================${NC}"
        echo -e "${BLUE}開始執行任務 $i / 12${NC}"
        echo -e "${BLUE}=====================================${NC}"
        echo ""
        run_task $i
        
        # 短暫暫停
        sleep 2
    done
    
    echo ""
    echo -e "${GREEN}🎉 所有任務執行完成！${NC}"
}

# 主邏輯
main() {
    if [[ $# -eq 0 ]]; then
        show_tasks
        echo "使用方式:"
        echo "  $0 <task_number>     - 執行特定任務 (1-12)"
        echo "  $0 all               - 依序執行所有任務"
        echo "  $0 next              - 繼續下一個任務 (通用指令)"
        echo "  $0 list              - 顯示任務列表"
        exit 0
    fi
    
    case "$1" in
        list)
            show_tasks
            ;;
        all)
            run_all
            ;;
        next)
            continue_next
            ;;
        [1-9]|1[0-2])
            run_task "$1"
            ;;
        *)
            echo -e "${YELLOW}❌ 錯誤: 無效的參數 '$1'${NC}"
            echo "使用 '$0' 查看使用說明"
            exit 1
            ;;
    esac
}

main "$@"
