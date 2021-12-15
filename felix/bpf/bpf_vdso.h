// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The logic to get the kernel version code from vdso is same as 
// the one defined in bpftrace under the iovisor project. iovisor project 
// is licensed under Apache License, Version 2.0.

// Reference: https://github.com/iovisor/bpftrace/blob/master/src/utils.c

#include <stdbool.h>
#include <stdint.h>
#include <string.h>
#include <limits.h>
#include <elf.h>
#include <sys/auxv.h>
#include <sys/utsname.h>
#include <stdio.h>

#ifndef ELF_BITS
# if ULONG_MAX > 0xffffffffUL
#  define ELF_BITS 64
# else
#  define ELF_BITS 32
# endif
#endif

#define ELF_BITS_XFORM2(bits, x) Elf##bits##_##x
#define ELF_BITS_XFORM(bits, x) ELF_BITS_XFORM2(bits, x)
#define ELF(x) ELF_BITS_XFORM(ELF_BITS, x)

int get_version_from_vdso() {
	__u64 base = getauxval(AT_SYSINFO_EHDR);
	ELF(Ehdr) *hdr = (ELF(Ehdr)*)base;
	for (int i = 0; i < hdr->e_shnum; i++) {
		ELF(Shdr) *shdr = (ELF(Shdr)*)(base + hdr->e_shoff + (i*hdr->e_shentsize));
		if (shdr->sh_type == SHT_NOTE) {
			char *ptr = (char *)(base + shdr->sh_offset);
			char *end = ptr + shdr->sh_size;
			while (ptr < end) {
				ELF(Nhdr) *nhdr = (ELF(Nhdr)*)ptr;
				ptr += sizeof (*nhdr);

				char *name = ptr;
				ptr += (nhdr->n_namesz + sizeof(ELF(Word)) - 1) & -(sizeof(ELF(Word)));

				char *desc = ptr;
				ptr += (nhdr->n_descsz + sizeof(ELF(Word)) - 1) & -(sizeof(ELF(Word)));

				if ((nhdr->n_namesz > 5 && !memcmp(name, "Linux", 5)) &&
					nhdr->n_descsz == 4 && !nhdr->n_type) {
					return *(int*)desc;
				}
			}
		}
	}
	return 0;
}

