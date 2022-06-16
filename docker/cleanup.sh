#!/bin/bash
# Disable exit on non 0
set +e

# These packages are split up into chunks of dependent packages (more or less).
PACKAGES="
bzip2-libs
ca-certificates
crypto-policies-scripts
cryptsetup-libs
curl
cyrus-sasl-lib
dbus
dbus-common
dbus-daemon
dbus-glib
dbus-libs
dbus-tools
device-mapper
device-mapper-libs
dnf
elfutils-default-yama-scope
elfutils-libs
expat
file-libs
fontconfig
freetype
gawk
gdbm
glib2
gnupg2
gnutls
gobject-introspection
gpgme
ima-evm-utils
json-c
json-glib
kmod-libs
krb5-libs
libarchive
libblkid
libcomps
libcurl
libdb
libdb-utils
libdnf
libfdisk
libidn2
libmodulemd
libmount
libnsl2
libpng
libpsl
libpwquality
librepo
librhsm
libseccomp
libsemanage
libsmartcols
libsolv
libssh
libtasn1
libtirpc
libusbx
libuser
libuuid
libyaml
libzstd
lua-libs
lz4-libs
nss-3.67.0-7.el8_5.x86_64
nss-sysinit
openldap
openssl-libs
p11-kit
p11-kit-trust
pam
passwd
platform-python
platform-python-setuptools
python3-chardet
python3-cloud-what
python3-dateutil
python3-dbus
python3-decorator
python3-dmidecode
python3-dnf
python3-dnf-plugins-core
python3-ethtool
python3-gobject-base
python3-gpg
python3-hawkey
python3-idna
python3-iniparse
python3-inotify
python3-libcomps
python3-libdnf
python3-librepo
python3-libs
python3-libxml2
python3-pip-wheel
python3-pysocks
python3-requests
python3-rpm
python3-setuptools-wheel
python3-six
python3-subscription-manager-rhsm
python3-syspurpose
python3-urllib3
readline
rpm
rpm-build-libs
rpm-libs
shadow-utils
systemd
systemd-libs
systemd-pam
tar
tpm2-tss
usermode
util-linux
virt-what
yum
"


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
