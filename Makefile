proto:
	protoc --go_out=. --go-grpc_out=. ./protos/coder.proto
	protoc --go_out=. --go-grpc_out=. ./protos/service.proto
	python3 -m grpc_tools.protoc -I./protos --python_out=sample-python-client --grpc_python_out=sample-python-client ./protos/coder.proto
	python3 -m grpc_tools.protoc -I./protos --python_out=sample-python-client --grpc_python_out=sample-python-client ./protos/service.proto

clean: 
	rm -rf tmp coding

python:
	cp /Users/sejal/Personal/codexec/protos/coder.proto /Users/sejal/Work/techforce/supervity-agent-runtime-poetry/protos/coder.proto

build: clean
	go build	
