# Calico SELinux package

This folder containers Calico custom SELinux policy for Red Hat Enterprise Linux (RHEL) and its bug-for-bug compatible operating system. [CentOS Stream](https://www.centos.org/centos-stream/) is not a RHEL binary compatible operating system.

## Build

Run `make build` and RPM packages can be found under `build/dist`. `noarch` folder containers the final packages to be distributed and `source` folder containers the source RPM packages. Currently, only RHEL version 8 and 9 are supported.

## Update

Follow these steps to update the custom SELinux policies.

### Source files

Add or update rule definitions in the following files:

- `calico.fc`: file context expressions.
- `calico.if`: interface and template definitions.
- `calico.te`: type enforcement rules.
- `calico-selinux.spec`: RPM spec file for `rpmbuild`.

### Update minimum dependency versions in SPEC file

This work was originally for RKE2 on RHEL so the [rke2-selinux](https://github.com/rancher/rke2-selinux) package requirements should still be met. See comments in the SPEC file.

### Update changelog and version

Refer to the [Fedora Packaging Guidelines](https://docs.fedoraproject.org/en-US/packaging-guidelines/) on how to update [changelog](https://docs.fedoraproject.org/en-US/packaging-guidelines/#changelogs) and [version](https://docs.fedoraproject.org/en-US/packaging-guidelines/Versioning/).

Update `policy_module` version in `calico.te` to match SPEC file.

## Publish

### Checkout your release branch

```bash
git clone git@github.com:tigera/calico-private.git
git checkout release-calient-vX.Y
```

### Change to the selinux directory

```bash
cd selinux
```

### Clean the packages

```bash
make clean
```

### Generate the packages

```bash
make build
```

### Push the SELinux RPM packages

```bash
aws --profile helm s3 cp selinux/build/rhel8/dist/noarch/calico-selinux-x.y-z.el8.noarch.rpm s3://tigera-public/ee/archives/ --acl public-read
aws --profile helm s3 cp selinux/build/rhel9/dist/noarch/calico-selinux-x.y-z.el9.noarch.rpm s3://tigera-public/ee/archives/ --acl public-read
# any other RPM packages for newer RHEL releases
```
