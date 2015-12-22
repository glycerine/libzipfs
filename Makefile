.PHONY: all demo demo2

curdir = $(shell pwd)

# determine appropriate umount command, which depends on OS.
UMOUNT := fusermount -u

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    UMOUNT = fusermount -u
endif
ifeq ($(UNAME_S),Darwin)
    UMOUNT = umount
endif

all:
	cd cmd/libzipfs-combiner && go install
	cd cmd/mountzip && go install

demo:
	cd cmd/libzipfs-combiner && go install
	go build -o api-demo testfiles/api.go
	rm -f ./api-demo-combo
	libzipfs-combiner -exe ./api-demo -zip testfiles/hi.zip -o ./api-demo-combo
	./api-demo-combo


# This is not a great demo under a makefile, but still demonstrates 
# steps you would do manually during the process of inspecting your combo file
# Zip contents by mounting it with mountzip.
#
# Possible bad side effect: if you are running other mountzip, this will pkill them too.
demo2:
	cd cmd/mountzip && go install
	rmdir testfiles/mnt || true
	mkdir testfiles/mnt
	${GOPATH}/bin/mountzip -zip testfiles/expectedCombined -mnt testfiles/mnt &
	sleep 1
	# next line should output 'saluations'
	cat testfiles/mnt/dirA/dirB/hello
	diff testfiles/mnt/dirA/dirB/hello testfiles/expected.hello
	pkill mountzip
	sleep 1
	$(UMOUNT) ${curdir}/testfiles/mnt
	rmdir testfiles/mnt
