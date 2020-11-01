package common

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

/*
			logger.go

	Questo modulo si occupa della scrittura su di un log delle operazioni avvenute. Per motivi di usabilità
		il log avviene su un file e non sulla console.

*/


var initializedLog = false 			//Variabile per memorizzare se il log è gia stato inizializzato
var RemoteLogConnection net.Conn	//Connessione con il logger remoto

// Inizializza il Log
func initializeLog() (retErr error) {

	if initializedLog {
		return nil
	}
	initializedLog = true

	var err error

	//Istanzio la connessione con il remote logger
	RemoteLogConnection, err = net.Dial("tcp", Config.LoggerHost)
	if err != nil {
		fmt.Println("Errore in dial: " + err.Error())
	}
	err = SendMessage(RemoteLogConnection, "d\n")
	if err != nil {
		fmt.Println("Errore nell'inizializzazione della connessione: " + err.Error())
	}


	//Creo una cartella di log

	err = os.Mkdir("log", 0755)
	if err != nil {
		fmt.Println("Errore nella creazione della cartella \"log\"")
		fmt.Println(err.Error())
	}

	logFileName := "log_" + strconv.Itoa(time.Now().Year()) + "_" + strconv.Itoa(int(time.Now().Month())) + "_" + strconv.Itoa(time.Now().Day()) + "_" +
		strconv.Itoa(time.Now().Hour()) + "_" + strconv.Itoa(time.Now().Minute()) + "_" + strconv.Itoa(time.Now().Second()) + ".txt"

	//Inizializzo il file di log

	file, err := os.OpenFile("log/"+logFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Errore nella creazione della file di log")
		fmt.Println(err.Error())
		return err
	}

	log.SetOutput(file)

	return nil
}

//Scrive sul log Info generali
func Info(message string) {
	log.Println("[INFO] " + message)
	fmt.Println("[INFO] " + message)

}

//Scrive sul log Warning che potrebbero portare al malfunzionamento dell'applicativo
func Warning(message string) {
	log.Println("[WARN] " + message)
	fmt.Println("[WARN] " + message)
}

//Scrive sul log eventi Fatal che portano al crash dell'applicativo
func Fatal(message string) {
	log.Println("[FATAL] " + message)
	fmt.Println("[FATAL] " + message)
}
