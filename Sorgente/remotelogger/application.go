package main

import (
	"fmt"
	"net"
	"strconv"
)

/*
			remote-logger.go

	Questo modulo si occupa del logging remoto del sistema

*/

var initialized = false
var clientConnections []net.Conn

//Inizializza il log
func main(){ Main() }


func Main() {

	if initialized {
		return
	}
	initialized = true

	

	handleClientConnections()

}


func handleClientConnections() {

	//Aspetta connessione TCP sulla porta tcp_port
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(60001))
	if err != nil {
		fmt.Println("Errore nell'apertura della connessione TCP")
		return
	}
	fmt.Println("Server per invio del log in ascolto")

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Errore nell'instaurazione della connessione")
		} else {

			go handleConnection(connection)
		}

	}
}

func handleConnection(connection net.Conn){

//Connessione in modalità listen
			msg, err := ReadMessage(connection)
			if err == nil {
				if msg == "l\n"{
					clientConnections = append(clientConnections, connection)
					_, _ = connection.Write([]byte("\n\t\t+-------------------------------+\n\t\t|                               |\n\t\t| Sistema per il logging remoto |\n\t\t|                               |\n\t\t+-------------------------------+\n\nConnessione effettuata\n"))

				} else {
					handleSystemMessages(connection)
				}
			} else { fmt.Println("Connessione rifiutata") }
}


func handleSystemMessages(connection net.Conn) {

	for {

		msg, err := ReadMessage(connection)
		if err != nil { return }
		err = SendMessage(msg)
		if err != nil {
			fmt.Println("Errore nell'invio del messaggio")
			return
		}
	}

}

func ReadMessage(connection net.Conn) (message string, retErr error){

	var bytes = make([]byte, 1024)

	_, err := connection.Read(bytes)
	if err != nil {
		fmt.Println("Errore nella lettura del messaggio da " + connection.RemoteAddr().String() + ". " + err.Error())
		_ = connection.Close()
		return "", err

	} else {
		return ByteToString(bytes), nil
	}

}

//Funzione per l'invio di messaggi
func SendMessage(message string) (retErr error) {

	for i, connection := range clientConnections {
		_, err := connection.Write([]byte(message))
		if err != nil {
			fmt.Println("Errore nell'invio di messaggio \"" + message + "\" a " + connection.RemoteAddr().String() + ". " + err.Error())
			clientConnections = append(clientConnections[:i], clientConnections[i+1:]...)
			connection.Close()
			
		}
	}

	return nil
}

//Funzione che converte da array di byte a stringa
func ByteToString(bytes []byte) string {

	n := -1
	for i, b := range bytes {
		if b == 0 {
			break
		}
		n = i
	}
	return string(bytes[:n+1])
}
