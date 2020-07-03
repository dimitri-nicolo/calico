#save paths for the packages we want to manually cleanup.
packagefiles=()
for OUTPUT in $(rpm -ql elfutils-libelf elfutils-libelf elfutils-default-yama-scope lua libdb dbus-libs dbus-python dbus dbus-glib bzip2-libs rpm rpm-python rpm-libs rpm-build-libs yum-metadata-parser yum yum-plugin-ovl yum-plugin-fastestmirror yum-utils)
do
	packagefiles+=($OUTPUT)
done


#force remove system packages
#elastic readiness probe shell script needs curl and bash.
rpm -e tar file-libs libdb-utils procps-ng libssh2 libsmartcols krb5-libs libblkid libuuid libmount util-linux python python-libs glib2 binutils lz4 libxml2 libxml2-python readline elfutils-libs nss-sysinit nss-tools expat vim-minimal elfutils-default-yama-scope ncurses-base openldap libidn gnupg2 gpgme libpng libtasn1 json-c systemd-libs freetype shadow-utils bind-license systemd sqlite --nodeps

#cleanup yum traces for libs
#packages are installed using yum
for i in tar file-libs libdb-utils libssh2 libsmartcols krb5-libs libblkid libuuid libmount util-linux python python-libs glib2 binutils lz4 libxml2 libxml2-python readline elfutils-libs nss-sysinit nss-tools expat vim-minimal elfutils-default-yama-scope ncurses-base ncurses-libs openldap libidn gnupg2 gpgme libpng libtasn1 json-c systemd-libs sqlite freetype shadow-utils bind-license systemd nss libdb dbus elfutils-libelf bzip2-libs lua rpm nspr pcre2

do
	#remove directory traces
	find /var/lib/yum/ -depth -name "*$i*" -type d -exec rm -r "{}" \;
done

#cleanup binaries which can not be removed by rpm
echo 'Removing related package paths:'
for i in "${packagefiles[@]}"
do
	echo $i
	rm -rf $i
done

#remove this script from fs
rm "$0"
