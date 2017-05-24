.PHONY: all test ut update-vendor

default: all
all: test
test: ut

vendor:
	glide install --strip-vendor

update-vendor:
	glide up --strip-vendor

ut:
	./run-uts

.PHONY: force
force:
	true

bin/calicoq: vendor protobuf force
	mkdir -p bin
	go build -o "$@" "./calicoq/calicoq.go"

release/calicoq: force
	mkdir -p release
	cd build-container && docker build -t calicoq-build .
	docker run --rm -v `pwd`:/calicoq calicoq-build /calicoq/build-container/build.sh

# Generate the protobuf bindings for Felix.
.PHONY: protobuf
protobuf: force vendor/github.com/projectcalico/felix/proto/felixbackend.pb.go
	true
vendor/github.com/projectcalico/felix/proto/felixbackend.pb.go: vendor/github.com/projectcalico/felix/proto/felixbackend.proto
	docker run --rm -v `pwd`/vendor/github.com/projectcalico/felix/proto:/src:rw \
	              calico/protoc \
	              --gogofaster_out=. \
	              felixbackend.proto

clean:
	-rm -f *.created
	find . -name '*.pyc' -exec rm -f {} +
	-rm -rf build bin release
	-docker rm -f calico-build
	-docker rmi calico/build
