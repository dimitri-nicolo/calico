#!/bin/bash
# Disable exit on non 0
set +e

# Remove python and python dependencies

PACKAGES="python3-libs platform-python python3-libcomps platform-python-setuptools platform-python-pip python3-rpm"
PACKAGES+=" python3-unbound python3-dnf python3-hawkey python3-libdnf python3-gpg shared-mime-info"
PACKAGES+=" crypto-policies-scripts unbound-libs dnf libdnf yum"

# delete systemd and dependent packages
PACKAGES+=" systemd-udev systemd-pam systemd dracut-squash dracut-network dracut dbus trousers-lib trousers kexec-tools"
PACKAGES+=" dhcp-client libkcapi-hmaccalc libkcapi dhcp-client iputils device-mapper-libs device-mapper os-prober grub2-tools"
PACKAGES+=" cryptsetup-libs kpartx grub2-tools grub2-tools-minimal grubby rpm-build-libs rpm-plugin-systemd-inhibit"

PACKAGES+=" kmod bind-export-libs kmod-libs openldap libevent ima-evm-utils xz openssl openssl-pkcs11 libidn2 gnupg2 gnutls"
PACKAGES+=" gnupg2 gpgme gnupg2-smime glib2 libmodulemd1 pinentry libsecret librepo elfutils-libs elfutils-debuginfod-client libsolv"
PACKAGES+=" libmodulemd shadow-utils libsemanage zip unzip libsolv gettext-libs gettext libcroco nmap-ncat json-c cyrus-sasl-lib util-linux"
PACKAGES+=" libpwquality kbd pam libnsl2 libtirpc iproute sqlite-libs elfutils-default-yama-scope tar file file-libs"
PACKAGES+=" procps-ng libsmartcols dbus-libs dbus-tools dbus-daemon systemd-libs dhcp-libs libusbx libblkid libuuid libmount"
PACKAGES+=" libfdisk binutils vim-minimal libyaml libseccomp tpm2-tss pciutils rdma-core libibverbs libpcap"
PACKAGES+=" iptables-libs nmap-ncat"

# remove packages vulnerable packages that rpm relies on last
PACKAGES+=" squashfs-tools libtasn1 lz4-libs lua-libs elfutils-libelf expat libcomps libmetalink readline gawk gdbm"
PACKAGES+=" p11-kit p11-kit-trust ca-certificates bzip2-libs ca-certificates libzstd krb5-libs openssl-libs"
PACKAGES+=" libcurl-minimal libarchive libdb libdb-utils curl libxml2 libcomps rpm-libs rpm"

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
