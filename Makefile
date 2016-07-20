.PHONEY: all test ut update-vendor

default: all
all: test
test: ut

update-vendor:
	glide up

ut:
	./run-uts

.PHONEY: force
force:
	true

bin/calicoq: force
	mkdir -p bin
	go build -o "$@" "./calicoq/main.go"

#release/calicoq: force
#	mkdir -p release
#	cd build-calicoctl && docker build -t calicoctl-build .
#	docker run --rm -v `pwd`:/libcalico-go calicoctl-build /libcalico-go/build-calicoctl/build.sh

clean:
	-rm -f *.created
	find . -name '*.pyc' -exec rm -f {} +
	-rm -rf build bin release
	-docker rm -f calico-build
	-docker rmi calico/build

