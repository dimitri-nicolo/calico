// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.

#include <linux/in.h>
#include "events_kprobe.h"
#include "kprobe.h"
#include <bpf_helpers.h>
#include <bpf_tracing.h>

SEC("kprobe/__x64_sys_execve")
int BPF_KPROBE(__x64_sys_execve)
{
	char *fileAddr = NULL, **argAddr = NULL;
	char *argp;
	int zero = 0;

	if (!ctx) {
		return 0;
	}
	
	/* The data read from this kprobes is 1420 bytes. With only
	 * 512 bytes of stack space, available for a BPF program,
	 * BPF_PER_CPU_ARRAY is used as a scratch space. The per-cpu array
	 * has only one element at index 0. Logic here is to read filename,
	 * args directly into the 0th element, write it to the LRU_HASH. This
	 * avoids the use of stack.
	 */
	struct calico_exec_value *data = cali_exec_lookup_elem(&zero);
	if (data) {
		__builtin_memset(data, 0, sizeof(struct calico_exec_value));
		/* x86 has syscall wrappers, as a result unwrap the ctx.
		 */
		struct pt_regs *__ctx = (struct pt_regs *)(PT_REGS_PARM1(ctx));
		// Read the address where filename is stored.
		bpf_probe_read(&fileAddr, sizeof(fileAddr), &PT_REGS_PARM1(__ctx));
		// Read the filename from fileAddr
		bpf_probe_read_str(&data->filename, sizeof(data->filename), (void*)fileAddr);
		// Read the address where the argument list is stored, into argAddr
		bpf_probe_read(&argAddr, sizeof(argAddr), &PT_REGS_PARM2(__ctx));
		#pragma unroll
		for (int i = 1; i < MAX_NUM_ARGS; i++) {
			// Read the address where each argument is stored.
			bpf_probe_read(&argp, sizeof(argp), (void*)&argAddr[i]);
			if (argp) {
				// Read the actual argument
				bpf_probe_read_str(&data->args[i-1], sizeof(data->args[i-1]), (void*)argp);
			} else {
				break;
			}
		}
		data->pid = bpf_get_current_pid_tgid() >> 32;
		data->hdr.type = EVENT_PROCESS_PATH;
		data->hdr.len = sizeof(struct calico_exec_value);
		cali_epath_update_elem(&data->pid, data, 0);
	}
        return 0;
}

char ____license[] __attribute__((section("license"), used)) = "GPL";

