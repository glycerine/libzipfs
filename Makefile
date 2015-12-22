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


# not great under a makefile, but demonstrates the process of inspecting your combo file
demo2: # osx only, may very well leave testfiles/mnt mounted at the end.
	cd cmd/mountzip && go install
	mkdir testfiles/mnt
	mountzip -zip testfiles/expectedCombined -mnt testfiles/mnt &
	sleep 1 && cat testfiles/mnt/dirA/dirB/hello
	pkill mountzip
	sleep 1
	umount ${curdir}/testfiles/mnt || fusermount -u ${curdir}/testfiles/mnt # on linux: fusermount -u instead.
	rmdir testfiles/mnt
