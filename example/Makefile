both: example1 example2 example3
example1: example.go
	go build -o $@ $^
example2: example2.go
	go build -o $@ $^
example3: example3.go
	go build -o $@ $^
clean: 
	bash -c "rm -f example{1,2,3}"
