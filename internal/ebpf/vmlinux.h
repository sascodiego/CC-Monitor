/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-008
 * CONTEXT:   Minimal vmlinux.h with essential kernel structures for eBPF programs
 * REASON:    Need kernel type definitions for syscall tracing without full BTF dependency
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep minimal and avoid conflicts with system headers
 * RISK:      Low - Minimal definitions reduce compatibility issues
 */

#ifndef __VMLINUX_H__
#define __VMLINUX_H__

typedef unsigned char __u8;
typedef short __s16;
typedef unsigned short __u16;
typedef int __s32;
typedef unsigned int __u32;
typedef long long __s64;
typedef unsigned long long __u64;
typedef __u8 u8;
typedef __s16 s16;
typedef __u16 u16;
typedef __s32 s32;
typedef __u32 u32;
typedef __s64 s64;
typedef __u64 u64;

#define AF_INET 2

struct sockaddr {
    __u16 sa_family;
    char sa_data[14];
};

struct in_addr {
    __u32 s_addr;
};

struct sockaddr_in {
    __u16 sin_family;
    __u16 sin_port;
    struct in_addr sin_addr;
    unsigned char sin_zero[8];
};

struct trace_event_raw_sys_enter {
    struct trace_entry ent;
    long id;
    unsigned long args[6];
    char __data[0];
};

struct trace_entry {
    unsigned short type;
    unsigned char flags;
    unsigned char preempt_count;
    int pid;
};

struct trace_event_raw_sched_process_template {
    struct trace_entry ent;
    char comm[16];
    __s32 pid;
    __s32 prio;
    __s32 exit_code;
    char __data[0];
};

struct task_struct {
    int pid;
    int tgid;
    char comm[16];
    struct task_struct *real_parent;
    __u32 uid;
    __u32 gid;
};

#endif /* __VMLINUX_H__ */