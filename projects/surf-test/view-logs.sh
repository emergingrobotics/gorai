#!/bin/bash
# Utility script to view and manage logs

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$SCRIPT_DIR/logs"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

show_help() {
    echo -e "${BLUE}Log Viewer for Surf Test${NC}"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  list              List all log files"
    echo "  latest [robot]    Show latest log (main-robot or ground-station)"
    echo "  tail [robot]      Tail latest log in real-time"
    echo "  search <pattern>  Search all logs for pattern"
    echo "  clean [days]      Remove logs older than N days (default: 7)"
    echo "  size              Show log directory size"
    echo ""
    echo "Examples:"
    echo "  $0 list"
    echo "  $0 latest main-robot"
    echo "  $0 tail ground-station"
    echo "  $0 search 'error'"
    echo "  $0 clean 3"
}

list_logs() {
    echo -e "${BLUE}Log Files:${NC}"
    echo ""
    if [ -d "$LOG_DIR" ] && [ "$(ls -A $LOG_DIR 2>/dev/null)" ]; then
        ls -lh "$LOG_DIR"/*.log 2>/dev/null | while read line; do
            echo "  $line"
        done
    else
        echo -e "${YELLOW}No log files found${NC}"
    fi
}

get_latest_log() {
    local prefix="$1"
    if [ -n "$prefix" ]; then
        ls -t "$LOG_DIR"/${prefix}*.log 2>/dev/null | head -1
    else
        ls -t "$LOG_DIR"/*.log 2>/dev/null | head -1
    fi
}

show_latest() {
    local prefix="$1"
    local latest=$(get_latest_log "$prefix")
    
    if [ -z "$latest" ]; then
        echo -e "${YELLOW}No log files found${NC}"
        return 1
    fi
    
    echo -e "${BLUE}Showing: ${CYAN}$latest${NC}"
    echo ""
    cat "$latest"
}

tail_latest() {
    local prefix="$1"
    local latest=$(get_latest_log "$prefix")
    
    if [ -z "$latest" ]; then
        echo -e "${YELLOW}No log files found${NC}"
        return 1
    fi
    
    echo -e "${BLUE}Tailing: ${CYAN}$latest${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
    echo ""
    tail -f "$latest"
}

search_logs() {
    local pattern="$1"
    
    if [ -z "$pattern" ]; then
        echo -e "${RED}Error: Search pattern required${NC}"
        return 1
    fi
    
    echo -e "${BLUE}Searching for: ${CYAN}$pattern${NC}"
    echo ""
    
    if [ -d "$LOG_DIR" ]; then
        grep -rn --color=always "$pattern" "$LOG_DIR"/*.log 2>/dev/null || echo -e "${YELLOW}No matches found${NC}"
    else
        echo -e "${YELLOW}Log directory not found${NC}"
    fi
}

clean_logs() {
    local days="${1:-7}"
    
    echo -e "${BLUE}Removing logs older than ${days} days...${NC}"
    
    if [ -d "$LOG_DIR" ]; then
        local count=$(find "$LOG_DIR" -name "*.log" -mtime +$days 2>/dev/null | wc -l)
        
        if [ "$count" -gt 0 ]; then
            find "$LOG_DIR" -name "*.log" -mtime +$days -delete 2>/dev/null
            echo -e "${GREEN}Removed $count log file(s)${NC}"
        else
            echo -e "${YELLOW}No old logs to remove${NC}"
        fi
    else
        echo -e "${YELLOW}Log directory not found${NC}"
    fi
}

show_size() {
    echo -e "${BLUE}Log Directory Size:${NC}"
    
    if [ -d "$LOG_DIR" ]; then
        du -sh "$LOG_DIR" 2>/dev/null
        echo ""
        echo -e "${BLUE}File count:${NC} $(ls -1 "$LOG_DIR"/*.log 2>/dev/null | wc -l) log files"
    else
        echo -e "${YELLOW}Log directory not found${NC}"
    fi
}

# Main
case "${1:-help}" in
    list)
        list_logs
        ;;
    latest)
        show_latest "$2"
        ;;
    tail)
        tail_latest "$2"
        ;;
    search)
        search_logs "$2"
        ;;
    clean)
        clean_logs "$2"
        ;;
    size)
        show_size
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac

