#core elastic dependancies are zlib, ncurses, bash, glibc, coreutils, libstdc++, libgcc
#force remove other system packages
rpm -e tar curl libcurl file-libs libdb-utils procps-ng libssh2 libsmartcols krb5-libs libblkid libuuid libmount util-linux python python-libs glib2 binutils lz4 libxml2 libxml2-python readline elfutils-libs nss-sysinit nss-tools expat vim-minimal elfutils-default-yama-scope ncurses-base openldap libidn gnupg2 gpgme libpng libtasn1 json-c  systemd-libs freetype shadow-utils bind-license systemd sqlite --nodeps

#cleanup yum traces for libs
#packages are installed using yum
for i in tar curl libcurl file-libs libdb-utils procps-ng libssh2 libsmartcols krb5-libs libblkid libuuid libmount util-linux python python-libs glib2 binutils lz4 libxml2 libxml2-python readline elfutils-libs nss-sysinit nss-tools expat vim-minimal elfutils-default-yama-scope ncurses-base openldap libidn gnupg2 gpgme libpng libtasn1 json-c systemd-libs sqlite freetype shadow-utils bind-license systemd nss libdb dbus elfutils-libelf bzip2-libs lua rpm nspr pcre2
do
	echo "$i"
	find /var/lib/yum/ -depth -name "*$i*" -type d -exec rm -r "{}" \;
done

#cleanup binaries which can not be removed by rpm
#clean manually
#lua
find /usr/lib64/ -depth -name "*lua*" -type f -exec rm -r "{}" \;
#libdb libdbus
find /usr/lib64/ -depth -name "*libdb*" -exec rm -r "{}" \;
#libelf
find /usr/lib64/ -depth -name "*libelf*" -exec rm -r "{}" \;
#bzip2-libs
find /usr/lib64/ -depth -name "*libbz2*" -exec rm -r "{}" \;
#nspr
find /usr/lib64/ -depth -name "*libnspr4*" -exec rm -r "{}" \;
#libcrypt
find /usr/lib64/ -depth -name "*libcrypt*" -exec rm -r "{}" \;

#remove this script from fs
rm "$0"
