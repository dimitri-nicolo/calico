# !/bin/bash
# save paths for the packages we want to manually cleanup.
packagefiles=('/var/log/yum.log')
for OUTPUT in $(rpm -ql elfutils-libelf elfutils-default-yama-scope lua libdb dbus-libs dbus-python dbus dbus-glib bzip2-libs rpm rpm-python rpm-libs rpm-build-libs yum-metadata-parser yum yum-plugin-ovl yum-plugin-fastestmirror yum-utils tar curl libcurl sqlite file-libs libdb-utils procps-ng libssh2 libsmartcols krb5-libs libblkid libuuid libmount util-linux python python-libs glib2 binutils lz4 libxml2 libxml2-python readline elfutils-libs nss-sysinit nss-tools expat vim-minimal elfutils-default-yama-scope ncurses-base openldap libidn gnupg2 gpgme libpng libtasn1 json-c systemd-libs freetype shadow-utils bind-license systemd)
do
        packagefiles+=($OUTPUT)
done

# cleanup binaries which can not be removed by rpm
echo 'Removing related package paths:'
for i in "${packagefiles[@]}"
do
        echo $i
        rm -rf $i
done

# remove this script from fs
rm "$0"
