#protoc -I=. --go_out=. --go-grpc_out=. --go-grpc_opt=paths=source_relative ./protos/messages/messages.proto
protoc -I=. --go_out=. --go-grpc_out=. --go-grpc_opt=paths=source_relative ./protos/services/services.proto
