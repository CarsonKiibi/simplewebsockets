package server

import (
	"fmt"
	"net/http"
	"net/url"
	"crypto/sha1"
	"encoding/base64"
)

var guid = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
var exampleKey = []byte("dGhlIHNhbXBsZSBub25jZQ==")

var allowedOrigins = []string{"localhost:2000"}

func generateAccept(key []byte) (string) {

	bytes := append(key, guid...)

	hasher := sha1.New()
	hasher.Write(bytes)
	encoded := base64.StdEncoding.EncodeToString(hasher.Sum(nil)) // url encoding makes + into - and / to _, protocol wants std
	fmt.Println("Accept header: ", encoded)
	return encoded
}

func sendServerHandshake(host string, path string, key []byte, subprotocols string) (error) {
	u := url.URL{Scheme: "http", Host: host, Path: path}

	req, err := http.NewRequest("GET", u.String(), nil) 
	if err != nil {
		fmt.Println("Error creating request: ", err)
		return err
	}

	accept := generateAccept(key)

	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Accept", accept)
	req.Header.Set("Sec-WebSocket-Protocol", subprotocols)

	return nil
}