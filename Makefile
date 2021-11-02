all:

run:
	go run main.go -addr 127.0.0.1:8989 --avatar-size 128000 --with-security-check

fake-key:
	go run server.go

.EXPORT_ALL_VARIABLES:
CODE_CORD_PATH=/home/artsem/Projects/GO/src/github.com/code-cord/cc.core.stream/release
CODE_CORD_SERVER_PUBLIC_KEY=/home/artsem/Downloads/id_rsa.pub
CODE_CORD_SERVER_PRIVATE_KEY=/home/artsem/Downloads/id_rsa
