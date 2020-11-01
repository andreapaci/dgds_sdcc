package main
import (
	"common"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

/*
			broker-configuration.go

	Questo modulo si occupa di recuperare e fornire i parametri di configurazione. Questi valori
		sono poi accessibili all'interno di tutto l'applicativo

*/

// Struct per la entry dei valori di configurazione sul DynamoDB
type ConfigEntry struct {
	FieldName  string //Nome del parametro
	FieldValue string //Valore del parametro

}

const configTable = "configuration" //Costante per il nome della tabella su DynamoDB


var delay_sqs_request	string	//Il tempo che intercorre tra una richiesta SQS ed un'altra
var delay_load_config	string	//Il tempo che intercorre per il fetch della nuova configurazione
var mq_threshold		float64	//Il limite massimo di persone al metro quadro
var positive_radius		int		//il raggio per mandare un messaggio quando si riscontra un positivo
var subTableName 		string  //Nome della tabella dove vengono gestite le sottoscrizioni
var globalSqsQueue 		string  //Nome della coda SQS usata dai broker per ricevere i messaggi


// Funzione che recupera le informazioni di configurazione dal database di DynamoDB
func retreiveConfig() (retErr error) {


	var configs []ConfigEntry //Variabile che memorizza i vari parametri di configurazione

	// Eseguo la query per recuperare i parametri
	configs, err := makeConfigQuery()
	if err != nil {
		common.Fatal("Errore nell'esecuzione della query\n" + err.Error())
		return err
	}

	// Assegno i parametri alle rispettive variabili
	err = assignParameters(configs)
	if err != nil {
		common.Fatal("Errore nell'associazione dei parametri\n" + err.Error())
		return err
	}

	return nil
}


// La funzione assegna i parametri ottenuti dalla query alle rispettive varabili
func assignParameters(configs []ConfigEntry) (retErr error) {

	var err error

	for _, conf := range configs {

		switch conf.FieldName {

			case "delay_sqs_request":
				delay_sqs_request = conf.FieldValue
			case "delay_load_config":
				delay_load_config = conf.FieldValue
			case "subTableName":
				subTableName = conf.FieldValue
			case "positive_radius":
				positive_radius, err = strconv.Atoi(conf.FieldValue)
				if err != nil {
					common.Fatal("[BROKER] Errore nel parsing del POSITIVE_RADIUS value, interruzione del programma\n" + err.Error())
					return err
				}
			case "mq_threshold":
				mq_threshold, err = strconv.ParseFloat(conf.FieldValue, 64)
				if err != nil {
					common.Fatal("[BROKER] Errore nel parsing del MQ_THRESHOLD value, interruzione del programma\n" + err.Error())
					return err
				}
			case "globalSqsQueue":
				globalSqsQueue = conf.FieldValue
			default:
				common.Fatal("La entry " + conf.FieldName + " non è valida, termino il programma")
				return errors.New("entry inesistente")
		}

	}

	if globalSqsQueue == "none"{
		globalSqsQueue, _ = createQueue("broker-reiceive")
		_ = updateConfigurationParameter("globalSqsQueue", globalSqsQueue)
	}

	//Controllo se tutte le variabili di configurazione sono corrette
	common.Info("[BROKER] Parametri ottenuti")
	common.Info(" |   Variabile delay_sqs_request "		+ delay_sqs_request 				+ " : " + reflect.TypeOf(delay_sqs_request).String())
	common.Info(" |   Variabile delay_load_config "		+ delay_load_config 				+ " : " + reflect.TypeOf(delay_load_config).String())
	common.Info(" |   Variabile subTableName " 			+ subTableName 						+ " : " + reflect.TypeOf(subTableName).String())
	common.Info(" |   Variabile mq_threshold " 			+ fmt.Sprintf("%f", mq_threshold) 	+ " : " + reflect.TypeOf(mq_threshold).String())
	common.Info(" |   Variabile positive_radius "		+ strconv.Itoa(positive_radius)		+ " : " + reflect.TypeOf(positive_radius).String())
	common.Info(" |   Variabile globalSqsQueue " 		+ globalSqsQueue 					+ " : " + reflect.TypeOf(globalSqsQueue).String())
	common.Info(" +-------------------------------------------------------------------------------------------------------\n\n")

	return nil
}
