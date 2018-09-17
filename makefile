all:
	@rm diamond.s 2>/dev/null || true # remove socket preventing build
	go build example.go
	go build admin.go
