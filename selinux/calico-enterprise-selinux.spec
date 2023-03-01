# Copyright (c) 2023 Tigera, Inc. All rights reserved.

%define calico_relabel_files() \
mkdir -p /var/lib/calico; \
mkdir -p /var/log/calico; \
mkdir -p /var/run/calico; \
mkdir -p /mnt/tigera; \
restorecon -FR /var/lib/calico; \
restorecon -FR /var/log/calico; \
restorecon -FR /var/run/calico; \
restorecon -FR /mnt/tigera

# The following versions need to be consistent with rke2-selinux versions.
# see https://github.com/rancher/rke2-selinux/blob/master/policy/centos8/rke2-selinux.spec#L20
%define selinux_policyver 3.13.1-252
%define container_policyver 2.167.0-1
%define container_policy_epoch 2

Name:       calico-enterprise-selinux
Version:    1.0
Release:    1%{?dist}
Summary:    SELinux policy module for Calico Enterprise

Group:      System Environment/Base
License:    Proprietary
URL:        https://tigera.io
Source0:    calico-enterprise.pp
Source1:    calico-enterprise.if

Requires: policycoreutils, libselinux-utils
Requires(post): selinux-policy-base >= %{selinux_policyver}
Requires(post): policycoreutils
Requires(post): container-selinux >= %{container_policy_epoch}:%{container_policyver}
Requires(postun): policycoreutils
BuildArch: noarch

Provides: %{name} = %{version}-%{release}

%description
This package installs and sets up the SELinux policy security module for Calico Enterprise.

%install
install -d %{buildroot}%{_datadir}/selinux/packages
install -m 644 %{SOURCE0} %{buildroot}%{_datadir}/selinux/packages
install -d %{buildroot}%{_datadir}/selinux/devel/include/contrib
install -m 644 %{SOURCE1} %{buildroot}%{_datadir}/selinux/devel/include/contrib/
install -d %{buildroot}/etc/selinux/targeted/contexts/users/

%pre
%selinux_relabel_pre

%post
semodule -n -i %{_datadir}/selinux/packages/calico-enterprise.pp
if /usr/sbin/selinuxenabled ; then
    /usr/sbin/load_policy
    %calico_relabel_files
fi;

%postun
if [ $1 -eq 0 ]; then
    %selinux_modules_uninstall calico-enterprise
fi;

%posttrans
%selinux_relabel_post

%files
%attr(0600,root,root) %{_datadir}/selinux/packages/calico-enterprise.pp
%{_datadir}/selinux/devel/include/contrib/calico-enterprise.if

%changelog
* Fri Jan 13 2023 Jiawei Huang <jiawei@tigera.io> - 1.0-1
- Initial release
