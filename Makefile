all: dit-cli dit-mirror

dit-cli:
	env GOOS=linux GOARCH=amd64 go build -o releases/linux/amd64/dit cmd/dit-cli/main.go
	env GOOS=linux GOARCH=arm64 go build -o releases/linux/arm64/dit cmd/dit-cli/main.go
	env GOOS=darwin GOARCH=amd64 go build -o releases/darwin/amd64/dit cmd/dit-cli/main.go
	env GOOS=darwin GOARCH=arm64 go build -o releases/darwin/arm64/dit cmd/dit-cli/main.go
	env GOOS=windows GOARCH=amd64 go build -o releases/windows/amd64/dit.exe cmd/dit-cli/main.go

dit-mirror:
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build --tags "linux" -o releases/linux/amd64/dit-mirror cmd/dit-mirror/main.go
	env GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build --tags "linux" -o releases/linux/arm64/dit-mirror cmd/dit-mirror/main.go
	env GOOS=darwin GOARCH=amd64 go build --tags "darwin" -o releases/darwin/amd64/dit-mirror cmd/dit-mirror/main.go
	env GOOS=darwin GOARCH=arm64 go build --tags "darwin" -o releases/darwin/arm64/dit-mirror cmd/dit-mirror/main.go
	env GOOS=windows GOARCH=amd64 go build -o releases/windows/amd64/dit-mirror.exe cmd/dit-mirror/main.go
	
