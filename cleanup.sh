#force remove system packages
#Readiness probe needs `curl libcurl libcrypt libnspr`
#sysctl needs systemd-libs procps-ng libelf libz/bz2 pcre systemd libcap
rpm -e tar file-libs libdb-utils libssh2 libsmartcols krb5-libs libblkid libuuid libmount util-linux python python-libs glib2 binutils lz4 libxml2 libxml2-python readline elfutils-libs nss-sysinit nss-tools expat vim-minimal elfutils-default-yama-scope ncurses-base openldap libidn gnupg2 gpgme libpng libtasn1 json-c freetype shadow-utils bind-license sqlite --nodeps

#cleanup yum traces for libs
#packages are installed using yum
for i in tar file-libs libdb-utils libssh2 libsmartcols krb5-libs libblkid libuuid libmount util-linux python python-libs glib2 binutils lz4 libxml2 libxml2-python readline elfutils-libs nss-sysinit nss-tools expat vim-minimal elfutils-default-yama-scope ncurses-base ncurses-libs openldap libidn gnupg2 gpgme libpng libtasn1 json-c sqlite freetype shadow-utils bind-license nss libdb dbus elfutils-libelf bzip2-libs lua rpm nspr pcre2

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

#remove this script from fs
rm "$0"
