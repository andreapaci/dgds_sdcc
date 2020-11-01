package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

/*
			common.go

	Questo modulo fornisce variabili e funzioni di utility che vengono condivise con tutto l'applicativo

*/

var Sess *session.Session                    //Variabile contenete i metadati della sessione con AWS

var Topics = []string{						//Topic preimpostati
	"Ristorazione",
	"Take-away",
	"Sanità",
	"Intrattenimento",
	"Circoli_sportivi",
	"Abbigliamento",
	"Cura_persona",
	"Farmacia",
	"Elettronica",
	"Uffici",
}

//File di configurazione locale
type LocalConfig struct {
	LoggerHost 		string
	AwsBroker 		string
	RetryDelay  	int
	PositDelay  	int
	OpDelay  		int
	SimulationTime	int
	RcvMessDelay	int
	MaxRcvMessage	int64
	PollingTime		int64
	Region			string
}

var Config LocalConfig

var TestPositionSize int = 30			//Lato dell'area geografica utilizzata per i test (generazione randomica)

// Struct per la entry dei valori di configurazione sul DynamoDB
type ConfigUpdateRequest struct {
	FieldName  string //Nome del parametro
	FieldValue string //Valore del parametro

}

//Le seguenti struct sono utilizzate come input di richiesta/risposta per le chiamate ad API REST
type SubRegistrationResponse struct {
	SubID     	string
	QueueURL  	string
}

type PubRegistrationResponse struct {
	QueueURL  	string
}

type SubPositionUpdateRequest struct {
	PositionX  	string
	PositionY  	string
}

type SubTopicSubscribeRequest struct {
	Topics []string
}

// Struct per la entry del subscriber/consumer sul DynamoDB
type SubscriberEntry struct {
	SubID     	string
	QueueURL  	string
	Topics    	[]string
	PositionX 	int
	PositionY 	int
}

//Inizializza l'ambiente
func InitializeEnvironment() (retError error) {

	//Leggo file di configurazione
	if readLocalConfig() != nil {
		Fatal("Errore nella lettura della configurazione locale")
		return errors.New("error loading local configuration")
	}

	//Inizializzazione del log
	if initializeLog() != nil {
		fmt.Println("Errore nell'inizializzazione del logger.")
		return
	}


	//Creazione di parametri di sessione
	var err error = nil
	Sess, err = session.NewSession(&aws.Config{
		Region:      aws.String(Config.Region),
	})
	if err != nil {
		Fatal("Errore nella instaurazionde della connessione con AWS\n" + err.Error())
		return err
	}


	Info("Parametri per la connessione con AWS creati.")

	return nil
}

//Legge il file di configurazione locale
func readLocalConfig() (retErr error) {

	jsonFile, err := os.Open("config.json")
	if err != nil {
		Fatal("Errore nell'ottenimento della configurazione locale. " + err.Error())
		return err
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &Config)
	if err != nil {
		Fatal("Errore nell'unmarshaling della configurazione locale. " + err.Error())
		return err
	}

	Info(fmt.Sprintf("Configurazione Locale: \n%+v\n", Config))

	return nil
}

//Legge input da tastiera per i modelli interattivi di publisher e subscriber
func ReadInput(reader *bufio.Reader) (in string, retErr error){

	var err error = nil
	var input string

	for {
		input, err = reader.ReadString('\n')
		if err != nil {
			Fatal(" Errore nella lettura dell'input. " + err.Error())
		} else { break }
	}

	if runtime.GOOS == "windows" {
		input = strings.Replace(input, "\n", "", -1)

	} else {
		input = strings.Replace(input, "\n", "", -1)
	}

	return input, nil
}

//Effettua una richiesta di tipo GET
func GetRequest(resousce string, output interface{}) (responseCode int, r interface{}, retErr error) {


	getResponse, err := http.Get("http://" + resousce)
	if err != nil {
		Fatal("Errore nella richiesta Get. " + err.Error())
		return 0, nil, err
	}

	return readResponse(getResponse, output)


}

//Effettua una richiesta di tipo POST
func PostRequest(resource string, input interface{}, output interface{}) (responseCode int, r interface{}, retErr error) {

	var jsonInput []byte
	var err error

	client := &http.Client{}

	if input != nil {
		jsonInput, err = json.Marshal(input)
		if err != nil {
			Fatal("Errore nel marshalling del input in JSON. " + err.Error())
			return 0, nil, err
		}
	}

	request, err := http.NewRequest(http.MethodPost, "http://"+resource, bytes.NewBuffer(jsonInput))
	if err != nil {
		Fatal("Errore nella creazione della richiesta POST JSON. " + err.Error())
		return 0, nil, err
	}

	request.Header.Set("Content-Type", "text/plain; charset=utf-8")

	postResponse, err := client.Do(request)
	if err != nil {
		Fatal("Errore nell'esecuzione della richiesta POST JSON. " + err.Error())
		return 0, nil, err
	}

	return readResponse(postResponse, output)
}

//Effettua una richiesta di tipo PUT
func PutRequest(resource string, input interface{}, output interface{}) (responseCode int, r interface{}, retErr error) {

	var jsonInput []byte
	var err error

	client := &http.Client{}

	if input != nil {
		jsonInput, err = json.Marshal(input)
		if err != nil {
			Fatal("Errore nel marshalling del input in JSON. " + err.Error())
			return 0, nil, err
		}
	}

	request, err := http.NewRequest(http.MethodPut, "http://" + resource, bytes.NewBuffer(jsonInput))
	if err != nil {
		Fatal("Errore nella creazione della richiesta PUT JSON. " + err.Error())
		return 0, nil, err
	}

	request.Header.Set("Content-Type", "text/plain; charset=utf-8")

	putResponse, err := client.Do(request)
	if err != nil {
		Fatal("Errore nell'esecuzione della richiesta PUT JSON. " + err.Error())
		return 0, nil, err
	}

	return readResponse(putResponse, output)

}

//Effettua una richiesta di tipo DELETE
func DeleteRequest(resource string, input interface{}, output interface{}) (responseCode int, r interface{}, retErr error) {

	var jsonInput []byte
	var err error

	client := &http.Client{}

	if input != nil {
		jsonInput, err = json.Marshal(input)
		if err != nil {
			Fatal("Errore nel marshalling del input in JSON. " + err.Error())
			return 0, nil, err
		}
	}

	request, err := http.NewRequest(http.MethodDelete, "http://"+resource, bytes.NewBuffer(jsonInput))
	if err != nil {
		Fatal("Errore nella creazione della richiesta DELETE JSON. " + err.Error())
		return 0, nil, err
	}
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")

	deleteResponse, err := client.Do(request)
	if err != nil {
		Fatal("Errore nell'esecuzione della richiesta DELETE JSON. " + err.Error())
		return 0, nil, err
	}

	return readResponse(deleteResponse, output)
}


//Funzione per estrapolare la risposta da una richiesta di tipo GET POST PUT DELETE
func readResponse(response *http.Response, output interface{}) (responseCode int, r interface{}, retErr error){

	if output == nil { return response.StatusCode, nil, nil}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		Fatal("Errore nella lettura del body della response. " + err.Error())
		return response.StatusCode, nil, err
	}



	err = json.Unmarshal(body, &output)
	if err != nil {
		Fatal("Errore nell'unmarshaling del body. " + err.Error())
		return response.StatusCode, nil, err
	}

	return response.StatusCode, output, nil

}


//Funzione per l'invio di messaggi con TCP
func SendMessage(connection net.Conn, message string) (retErr error) {

	if len(message) >= 1024 {
		Warning("Size del messaggio maggiore di 1024, il messaggio viene tagliato")
		return nil
	}

	if connection != nil {
		_, err := connection.Write([]byte(message))
		if err != nil {
			Fatal("Errore nell'invio di messaggio \"" + message + "\" a " + connection.RemoteAddr().String() + ". " + err.Error())
			return err
		}
	}

	return nil
}

//Funzione per la ricezione di messaggi con TCP
func RecvMessage(connection net.Conn) (message string, retErr error) {

	var bytes = make([]byte, 1024)

	_, err := connection.Read(bytes)
	if err != nil {
		Fatal("Errore nella lettura del messaggio da " + connection.RemoteAddr().String() + ". " + err.Error())
		return "", err
	}

	msg := ByteToString(bytes)

	return msg, nil

}

//Ritorna se una lista contiene una determinata stringa
func StringListContains(list []string, e string) (contain bool) {
	for _, t := range list {
		if t == e {
			return true
		}
	}
	return false
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

//Genera un SUB ID a partire da una lista di sub id già presente nel sistema
func GenerateSubID(subsID []string, startFrom int) (subID string, retErr error) {

	for i := startFrom; i > -1; i++ {
		if StringListContains(subsID, strconv.Itoa(i)) == false {
			return strconv.Itoa(i), nil
		}
	}

	return "", errors.New("no id for subscriber found")
}

//Concatena array di stringhe in una sola stringa usando un certo separatore
func ConcatenateArrayValues(arr []string, char string) (concat string) {

	str := ""
	for i, elem := range arr {
		if i < len(arr)-1 {
			str += elem + char
		} else {
			str += elem
		}
	}

	return str

}

//Funzioni di utility per convertire una risposta generale JSON in una struct specifica
func SetField(obj interface{}, name string, value interface{}) error {
	structValue := reflect.ValueOf(obj).Elem()
	structFieldValue := structValue.FieldByName(name)

	if !structFieldValue.IsValid() {
		return fmt.Errorf("No such field: %s in obj", name)
	}

	if !structFieldValue.CanSet() {
		return fmt.Errorf("Cannot set %s field value", name)
	}

	structFieldType := structFieldValue.Type()
	val := reflect.ValueOf(value)
	if structFieldType != val.Type() {
		return errors.New("Provided value type didn't match obj field type")
	}

	structFieldValue.Set(val)
	return nil
}


//
func FillStruct(s interface{}, m map[string]interface{}) error {
	for k, v := range m {
		err := SetField(s, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}
