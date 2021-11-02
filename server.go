package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang-jwt/jwt"
)

/*
import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"
)

type Listener int

type Reply struct {
	Data string
}

func (l *Listener) GetLine(line []byte, reply *Reply) error {
	rv := string(line)
	fmt.Printf("Receive: %v\n", rv)
	*reply = Reply{rv}
	return nil
}

func main() {
	addy, err := net.ResolveTCPAddr("tcp", "0.0.0.0:12345")
	if err != nil {
		log.Fatal(err)
	}
	inbound, err := net.ListenTCP("tcp", addy)
	if err != nil {
		log.Fatal(err)
	}
	listener := new(Listener)
	rpc.Register(listener)
	for {
		conn, err := inbound.Accept()
		if err != nil {
			continue
		}
		go func() {
			time.Sleep(time.Minute)
		}()
		jsonrpc.ServeConn(conn)
	}
}


func (s *Server) PPP() {
	client, err := jsonrpc.Dial("tcp", "0.0.0.0:30300")
	if err != nil {
		log.Fatal(err)
	}
	in := bufio.NewReader(os.Stdin)
	for {
		line, _, err := in.ReadLine()
		if err != nil {
			log.Fatal(err)
		}
		var reply Reply
		err = client.Call("Listener.GetLine", line, &reply)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Reply: %v, Data: %v", reply, reply.Data)
	}
}

type Reply struct {
	Data string
}

*/

func PrivateKeyToEncryptedPEM(bits int, pwd string) ([]byte, error) {
	// Generate the key of length bits
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}

	// Convert it to pem
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	// Encrypt the pem
	if pwd != "" {
		block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(pwd), x509.PEMCipherAES256)
		if err != nil {
			return nil, err
		}
	}

	return pem.EncodeToMemory(block), nil
}

func kek(data []byte) {
	block, rest := pem.Decode(data)
	if len(rest) > 0 {
		panic("extra data")
	}
	der, err := x509.DecryptPEMBlock(block, []byte("password"))
	if err != nil {
		panic("decrypt failed: " + err.Error())
	}
	if _, err := x509.ParsePKCS1PrivateKey(der); err != nil {
		panic("invalid private key: " + err.Error())
	}
	/*plainDER, err := base64.StdEncoding.DecodeString(data.plainDER)
	if err != nil {
		t.Fatal("cannot decode test DER data: ", err)
	}
	if !bytes.Equal(der, plainDER) {
		t.Error("data mismatch")
	}*/
}

func main() {
	/*data, _ := PrivateKeyToEncryptedPEM(512, "password")
	fmt.Println(string(data))

	kek(data)*/

	claims := &jwt.StandardClaims{
		ExpiresAt: time.Now().UTC().Add(10 * time.Hour).Unix(),
		Subject:   "azazaz",
	}

	privateKey, err := ioutil.ReadFile("/home/artsem/Downloads/id_rsa")
	if err != nil {
		panic(err)
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		panic(err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	str, err := token.SignedString(signKey)
	if err != nil {
		panic(err)
	}

	fmt.Println(str)
}
