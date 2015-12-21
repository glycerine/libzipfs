all:
	cd cmd/libzipfs-combiner && go install
	cd cmd/mountzip && go install

demo:
	cd cmd/libzipfs-combiner && go install
	go build -o api-demo testfiles/api.go
	rm -f ./api-demo-combo
	libzipfs-combiner -exe ./api-demo -zip testfiles/hi.zip -o ./api-demo-combo
	./api-demo-combo



