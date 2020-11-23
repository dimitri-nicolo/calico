# !/bin/bash
# save paths for the packages we want to manually cleanup.
packagefiles=('/var/log/yum.log')
for OUTPUT in $(rpm -ql elfutils-libelf elfutils-default-yama-scope lua libdb dbus-libs dbus-python dbus dbus-glib bzip2-libs rpm rpm-python rpm-libs rpm-build-libs yum-metadata-parser yum yum-plugin-ovl yum-plugin-fastestmirror yum-utils tar file-libs libdb-utils procps-ng libsmartcols libblkid libuuid libmount util-linux python python-libs glib2 binutils lz4 libxml2 libxml2-python readline elfutils-libs nss-sysinit nss-tools expat vim-minimal elfutils-default-yama-scope ncurses-base gnupg2 gpgme libtasn1 json-c systemd-libs shadow-utils bind-license systemd)
do
	packagefiles+=($OUTPUT)
done

# elastic readiness probe shell script needs curl and bash.
# Curl Requirements:
# libidn
# libssh2
# krb5-libs
# openldap
# eck-operator needs sqlite

# cleanup binaries which can not be removed by rpm
echo 'Removing related package paths:'
for i in "${packagefiles[@]}"
do
	echo $i
	rm -rf $i
done

# remove this script from fs
rm "$0"
