#!/bin/bash
# Disable exit on non 0
set +e

# Remove python and python dependencies

PACKAGES="bind-export-libs binutils bzip2-libs ca-certificates ca-certificates
crypto-policies-scripts cryptsetup-libs curl cyrus-sasl-lib dbus dbus-daemon
dbus-libs dbus-tools device-mapper device-mapper-libs dhcp-client dhcp-client
dhcp-libs dnf dracut dracut-network dracut-squash elfutils-debuginfod-client
elfutils-default-yama-scope elfutils-libelf elfutils-libs expat file file-libs
gawk gdbm gettext gettext-libs glib2 gnupg2 gnupg2 gnutls gpgme grub2-tools
grub2-tools-minimal grubby ima-evm-utils iproute iptables-libs iputils json-c
kbd kexec-tools kmod kmod-libs kpartx krb5-libs libarchive libblkid libbpf
libcomps libcomps libcroco libcurl-minimal libdb libdb-utils libdnf libevent
libfdisk libibverbs libidn2 libkcapi libkcapi-hmaccalc libmetalink libmodulemd
libmount libnsl2 libpcap libpwquality librepo libseccomp libsemanage
libsmartcols libsolv libsolv libtasn1 libtirpc libusbx libuuid libxml2 libyaml
libzstd lua-libs lz4-libs nmap-ncat nmap-ncat openldap openssl openssl-libs
openssl-pkcs11 os-prober p11-kit p11-kit-trust pam pciutils platform-python
platform-python-pip platform-python-setuptools procps-ng python3-dnf python3-gpg
python3-hawkey python3-libcomps python3-libdnf python3-libs python3-rpm
python3-unbound rdma-core readline rpm rpm-build-libs rpm-libs
rpm-plugin-systemd-inhibit shadow-utils shared-mime-info sqlite-libs
squashfs-tools systemd systemd-libs systemd-pam systemd-udev tar tpm2-tss
trousers trousers-lib unbound-libs unzip util-linux vim-minimal xz yum zip"

echo "$PACKAGES"

if ! PACKAGE_FILES=$(rpm -ql ${PACKAGES} | tr '\n' ' '); then
  echo "failed to list package files"
  exit 1
fi

if ! rpm -e ${PACKAGES}; then
  echo "failed to remove packages"
  exit 1
fi

# We don't care if rm fails, we're just making a best effort to remove any left over cruft from the erased packages.
rm -rf ${PACKAGE_FILES}
rm -rf /var/log/*

# Delete this script
rm "$0"

exit 0
