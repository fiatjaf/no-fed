no-fed: $(shell find . -name "*.go")
	go build -ldflags="-s -w" -o ./no-fed

deploy: no-fed
	ssh root@turgot 'systemctl stop no-fed'
	scp no-fed turgot:no-fed/no-fed
	ssh root@turgot 'systemctl start no-fed'
