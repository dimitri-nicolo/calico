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
	go build -o "$@" "./calicoq/calicoq.go"

release/calicoq: force
	mkdir -p release
	cd build-container && docker build -t calicoq-build .
	docker run --rm -v `pwd`:/calicoq calicoq-build /calicoq/build-container/build.sh

clean:
	-rm -f *.created
	find . -name '*.pyc' -exec rm -f {} +
	-rm -rf build bin release
	-docker rm -f calico-build
	-docker rmi calico/build

