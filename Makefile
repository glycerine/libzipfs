.PHONY: all

curdir = $(shell pwd)

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
	mkdir testfiles/mnt
	${GOPATH}/bin/mountzip -zip testfiles/expectedCombined -mnt testfiles/mnt &
	sleep 1
	cat testfiles/mnt/dirA/dirB/hello
	pkill mountzip
	sleep 1
	# the next line should use 'umount' on OSX, and 'fusermount -u' on linux
	umount ${curdir}/testfiles/mnt || fusermount -u ${curdir}/testfiles/mnt
	rmdir testfiles/mnt
