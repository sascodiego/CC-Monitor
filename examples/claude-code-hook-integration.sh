#!/bin/bash

/**
 * CONTEXT:   Claude Code hook integration script for daemon-managed session correlation
 * INPUT:     Claude Code hook events and command execution context
 * OUTPUT:    Session start/end events sent to daemon without temporary files or environment variables
 * BUSINESS:  Provide seamless integration with Claude Code using automatic context detection
 * CHANGE:    Initial hook integration script replacing file-based correlation system
 * RISK:      High - Hook integration accuracy affects all work hour tracking for Claude Code users
 */

# Configuration
CLAUDE_HOOK_BINARY="${CLAUDE_HOOK_BINARY:-./claude-hook}"
DAEMON_URL="${CLAUDE_DAEMON_URL:-http://localhost:8080}"
DEBUG="${CLAUDE_HOOK_DEBUG:-false}"
HOOK_TYPE="${1:-activity}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    if [ "$DEBUG" = "true" ]; then
        echo -e "${GREEN}[INFO]${NC} $1" >&2
    fi
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

/**
 * CONTEXT:   Check if daemon is available before sending hook events
 * INPUT:     Daemon URL for health check
 * OUTPUT:    Exit code 0 if daemon is healthy, non-zero if unavailable
 * BUSINESS:  Prevent hook failures when daemon is not running
 * CHANGE:    Initial daemon health check for robust hook integration
 * RISK:      Low - Health check prevents unnecessary hook failures
 */
check_daemon_health() {
    local health_url="${DAEMON_URL}/api/health"
    
    log_info "Checking daemon health at $health_url"
    
    if command -v curl >/dev/null 2>&1; then
        response=$(curl -s -f --max-time 3 "$health_url" 2>/dev/null)
        if [ $? -eq 0 ]; then
            log_info "Daemon is healthy"
            return 0
        fi
    elif command -v wget >/dev/null 2>&1; then
        response=$(wget -q --timeout=3 -O - "$health_url" 2>/dev/null)
        if [ $? -eq 0 ]; then
            log_info "Daemon is healthy"
            return 0
        fi
    fi
    
    log_warn "Daemon health check failed"
    return 1
}

/**
 * CONTEXT:   Execute hook command with automatic error handling and retry logic
 * INPUT:     Hook type and additional arguments for hook command
 * OUTPUT:    Hook execution result with proper error handling
 * BUSINESS:  Ensure reliable hook execution with fallback strategies
 * CHANGE:    Initial hook execution wrapper with error handling
 * RISK:      Medium - Hook execution reliability affects session tracking
 */
execute_hook() {
    local hook_type="$1"
    shift
    local additional_args="$@"
    
    log_info "Executing hook: type=$hook_type, args='$additional_args'"
    
    # Build hook command
    local hook_cmd="$CLAUDE_HOOK_BINARY"
    hook_cmd="$hook_cmd --type=$hook_type"
    hook_cmd="$hook_cmd --daemon-url=$DAEMON_URL"
    
    if [ "$DEBUG" = "true" ]; then
        hook_cmd="$hook_cmd --debug"
    fi
    
    # Add additional arguments
    if [ -n "$additional_args" ]; then
        hook_cmd="$hook_cmd $additional_args"
    fi
    
    log_info "Hook command: $hook_cmd"
    
    # Execute hook with timeout
    if timeout 30s $hook_cmd; then
        log_info "Hook executed successfully: $hook_type"
        return 0
    else
        local exit_code=$?
        log_error "Hook execution failed: $hook_type (exit code: $exit_code)"
        return $exit_code
    fi
}

/**
 * CONTEXT:   Claude Code pre-action hook for session start events
 * INPUT:     Claude command and execution context
 * OUTPUT:     Session start event sent to daemon
 * BUSINESS:  Start session tracking when Claude Code action begins
 * CHANGE:    Initial pre-action hook for session start tracking
 * RISK:      High - Session start accuracy affects all subsequent correlation
 */
claude_pre_action_hook() {
    log_info "Claude pre-action hook triggered"
    
    # Get command from environment or arguments
    local command="${CLAUDE_COMMAND:-$1}"
    
    # Check daemon availability
    if ! check_daemon_health; then
        log_warn "Daemon not available, skipping pre-action hook"
        return 0
    fi
    
    # Execute start hook
    execute_hook "start" --command="$command" --skip-on-failure
}

/**
 * CONTEXT:   Claude Code post-action hook for session end events
 * INPUT:     Claude command execution results including duration and token count
 * OUTPUT:    Session end event sent to daemon with processing metrics
 * BUSINESS:  Complete session tracking when Claude Code action finishes
 * CHANGE:    Initial post-action hook for session completion tracking
 * RISK:      High - Session end correlation accuracy critical for time tracking
 */
claude_post_action_hook() {
    log_info "Claude post-action hook triggered"
    
    # Parse arguments
    local command="${CLAUDE_COMMAND:-}"
    local duration="${CLAUDE_DURATION:-0}"
    local tokens="${CLAUDE_TOKENS:-0}"
    local success="${CLAUDE_SUCCESS:-true}"
    local error_message="${CLAUDE_ERROR:-}"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --command=*)
                command="${1#*=}"
                shift
                ;;
            --duration=*)
                duration="${1#*=}"
                shift
                ;;
            --tokens=*)
                tokens="${1#*=}"
                shift
                ;;
            --success=*)
                success="${1#*=}"
                shift
                ;;
            --error=*)
                error_message="${1#*=}"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check daemon availability
    if ! check_daemon_health; then
        log_warn "Daemon not available, skipping post-action hook"
        return 0
    fi
    
    # Build hook arguments
    local hook_args=""
    if [ -n "$duration" ] && [ "$duration" != "0" ]; then
        hook_args="$hook_args --duration=$duration"
    fi
    if [ -n "$tokens" ] && [ "$tokens" != "0" ]; then
        hook_args="$hook_args --tokens=$tokens"
    fi
    if [ "$success" = "false" ]; then
        hook_args="$hook_args --success=false"
        if [ -n "$error_message" ]; then
            hook_args="$hook_args --error=\"$error_message\""
        fi
    fi
    
    # Execute end hook
    execute_hook "end" $hook_args --skip-on-failure
}

/**
 * CONTEXT:   General activity hook for misc Claude Code events
 * INPUT:     Activity type and context information
 * OUTPUT:    Activity event sent to daemon for general tracking
 * BUSINESS:  Track general Claude Code usage patterns and activity
 * CHANGE:    Initial activity hook for general event tracking
 * RISK:      Low - General activity tracking for usage insights
 */
claude_activity_hook() {
    log_info "Claude activity hook triggered"
    
    local command="${CLAUDE_COMMAND:-$1}"
    
    # Check daemon availability (optional for activity hooks)
    if ! check_daemon_health; then
        log_info "Daemon not available, skipping activity hook"
        return 0
    fi
    
    # Execute activity hook
    execute_hook "activity" --command="$command"
}

# Main script logic
main() {
    log_info "Claude Code hook integration started: type=$HOOK_TYPE"
    
    case "$HOOK_TYPE" in
        "pre_action"|"start")
            claude_pre_action_hook "$@"
            ;;
        "post_action"|"end")
            claude_post_action_hook "$@"
            ;;
        "activity")
            claude_activity_hook "$@"
            ;;
        *)
            log_error "Unknown hook type: $HOOK_TYPE"
            echo "Usage: $0 {pre_action|post_action|activity} [options]"
            echo "Environment variables:"
            echo "  CLAUDE_COMMAND      - Command that triggered the hook"
            echo "  CLAUDE_DURATION     - Processing duration in seconds"
            echo "  CLAUDE_TOKENS       - Token count for the response"
            echo "  CLAUDE_SUCCESS      - Whether the operation was successful (true/false)"
            echo "  CLAUDE_ERROR        - Error message if success=false"
            echo "  CLAUDE_DAEMON_URL   - Daemon URL (default: http://localhost:8080)"
            echo "  CLAUDE_HOOK_DEBUG   - Enable debug output (true/false)"
            exit 1
            ;;
    esac
}

# Execute main function with all arguments
main "$@"