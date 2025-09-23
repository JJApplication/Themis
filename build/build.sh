echo "start build"
go clean
go build -trimpath -o themis ./cmd/server/main.go

echo "build success"