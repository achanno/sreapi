protoc --grpc-gateway_out=logtostderr=true:. protobuf/vm.proto -I"${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis"  -I .
protoc --go_out=plugins=grpc:. protobuf/vm.proto -I"${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis"  -I .
