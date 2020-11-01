package main

import (
	"common"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"net/http"
	"strconv"
	"time"
)

/*
			handle_tcp_request.go

	Questo modulo si occupa dell'interazione con publishers e subscriber da parte del broker. La comunicazione
		avviene attraverso API REST.
	Per fare ciò è necessario creare un server http (usando httpd e gorilla/mux)

*/




// Funzione che inizializza il server htttp e imposta il comportamendo da esegure per ogni tipo di richiesta da fare
func handleRequests(){

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", checkVital)
	router.HandleFunc("/configuration", getConfiguration).Methods("GET")
	router.HandleFunc("/configuration", updateConfiguration).Methods("POST")
	router.HandleFunc("/configuration", modifyConfiguration).Methods("PUT")
	router.HandleFunc("/subscriber", getSubscriber).Methods("GET")
	router.HandleFunc("/subscriber", handleSubscriberRegistration).Methods("PUT")
	router.HandleFunc("/publisher", handlePublisher).Methods("GET")
	router.HandleFunc("/subscriber/{id}/position", handlePositionUpdate).Methods("POST")
	router.HandleFunc("/subscriber/{id}/topic", handleTopicSubscribe).Methods("PUT")
	router.HandleFunc("/subscriber/{id}/topic", handleTopicUnsubscribe).Methods("DELETE")
	router.HandleFunc("/subscriber/{id}", handleSubscriberRemoval).Methods("DELETE")

	//Abilita il Cross origin request
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowCredentials: true,
		AllowedMethods: []string{"GET", "PUT", "POST", "DELETE"},
	})

	handler := c.Handler(router)
	//Ascolto sulla porta 80
	err := http.ListenAndServe(":80", handler)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'inizializzazione dell'API REST")
	}

}

//Funzione per rispondere ad una richiesta GET per il check alive
func checkVital(w http.ResponseWriter, r *http.Request) {
	_, err :=fmt.Fprintf(w, "Sistema in running.")
	if err != nil {
		common.Fatal("[BROKER] Errore nell' check del sistema")
	}


}


//Ottieni lista subscribers
func getSubscriber(w http.ResponseWriter, r *http.Request){

	common.Info("[BROKER] Comando fetch dei subscribers.")

	subs, err := getSubscribers()
	if err != nil {
		common.Fatal("[BROKER] Errore nell'ottenimento dei subscribers' " + err.Error())
		http.Error(w, "Error in fetching subscribers.\n" + err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(subs)
	if err != nil {
		common.Fatal("[BROKER] Errore nel marshalling dei subscribers. " + err.Error())
		http.Error(w, "Error in response marshalling.\n" + err.Error(), http.StatusInternalServerError)
		return
	}



}

//Ottieni lista configurazione
func getConfiguration(w http.ResponseWriter, r *http.Request){

	common.Info("[BROKER] Comando fetch dei parametri di configurazione")

	configs, err := makeConfigQuery()
	if err != nil {
		common.Fatal("[BROKER] Errore nell'ottenimento dei parametri' " + err.Error())
		http.Error(w, "Error in fetching configuration parameters.\n" + err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(configs)
	if err != nil {
		common.Fatal("[BROKER] Errore nel marshalling dei parametri di configurazione. " + err.Error())
		http.Error(w, "Error in response marshalling.\n" + err.Error(), http.StatusInternalServerError)
		return
	}



}


//Forza l'aggiornamento della configurazione del broker
func updateConfiguration(w http.ResponseWriter, r *http.Request){

	common.Info("[BROKER] Comando aggiornamento parametri di configurazione")
	err := retreiveConfig()
	if err != nil {
		common.Info("[BROKER] Errore nell'aggiornamento della configurazione. " + err.Error())
		http.Error(w, "Error in updating configuration parameters.\n" + err.Error(), http.StatusInternalServerError)
		return
	}
}

//Modifica parametro di configurazione
func modifyConfiguration(w http.ResponseWriter, r *http.Request){


	configModify := ConfigEntry{}
	err := json.NewDecoder(r.Body).Decode(&configModify)
	if err != nil {
		common.Fatal("[BROKER] Errore nel unmarshalling della richiesta. " + err.Error())
		http.Error(w, "Error in request marshalling.\n"+err.Error(), http.StatusBadRequest)
		return
	}

	common.Info("[BROKER] Comando modifica dei parametri di configurazione: " + configModify.FieldName + " = " + configModify.FieldValue)

	err = updateConfigurationParameter(configModify.FieldName, configModify.FieldValue)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'aggiornamento del parametro di configurazione. " + err.Error())
		http.Error(w, "Error in modifying a config value.\n"+err.Error(), http.StatusInternalServerError)
		return
	}

	//Conseguente update forzato del borker
	updateConfiguration(w, r)


}



//Funzione che gestisce la registrazione di un subscriber
func handleSubscriberRegistration(w http.ResponseWriter, r *http.Request) {

	common.Info("[BROKER] Comando registrazione subscriber")

	//Registro il nuovo subscriber

	subID, queueUrl, err := registerSubscriber()
	if err != nil {
		common.Fatal("[BROKER] Errore nella registrazione del subscriber. " + err.Error())
		http.Error(w, "Error in adding subscriber.\n" + err.Error(), http.StatusInternalServerError)
		return
	}

	common.Info("[BROKER] Registrazione del subscriber " + subID + " avvenuta con successo. Invio dei parametri.")

	err = json.NewEncoder(w).Encode(common.SubRegistrationResponse{SubID: subID, QueueURL: queueUrl})
	if err != nil {
		common.Fatal("[BROKER] Errore nel marshalling della risposta al subscriber. ( " + subID + ", " + queueUrl + "). " + err.Error())
		http.Error(w, "Error in response marshalling.\n" + err.Error(), http.StatusInternalServerError)
	}


}

//Effettua la registrazione su AWS del subscriber
func registerSubscriber() (ID string, queueURL string, retErr error) {

	//Registro il nuovo subscriber

	var queueUrl = ""
	var subID = "0"
	id, _ := strconv.Atoi(subID)

	subsID, err := getSubscribersID()
	if err != nil {
		common.Fatal("[BROKER] Errore nell'ottenimento della lista di subscribers")
		return "", "", err
	}

	//Dopo 3 tentativi la richiesta di generare un subscriber fallisce
	retry := 3
	for {

		subID, err = common.GenerateSubID(subsID, id)
		if err != nil {
			common.Fatal("[BROKER] Impossibile trovare ID valido per subscriber")
			return "", "", err
		}

		//Creazione di coda SQS

		queueUrl, err = createQueue(subID)
		if err != nil {
			common.Fatal("[BROKER] Errore nella creazione della coda della entry al DB\n" + err.Error())

			retry --
			if retry < 0 { return "", "", err}

			id, _ = strconv.Atoi(subID)
			id++

			time.Sleep(time.Second)

		} else { break }
	}


	//Aggiunge la entry al DB
	err, alreadyExisting := addEntryDB(subID, queueUrl)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'aggiunta del subscriber al DB\n" + err.Error())

		// Rimuovo la entry dal DB
		if alreadyExisting == false {

			//Se c'è stato un errore nell'aggiunta della entry al db, elimina anche la coda per evitare che rimanga una coda senza subscriber annesso
			if deleteQueue(queueUrl) != nil {
				common.Fatal("[BROKER] Errore nell'eliminazione della coda\n" + err.Error())
			}
		}


		return "", "", err
	}

	return subID, queueUrl, nil
}



//Gestisce la registrazione di un publisher
func handlePublisher(w http.ResponseWriter, r *http.Request) {

	common.Info("[BROKER] Comando registrazione publisher")

	err := json.NewEncoder(w).Encode(common.PubRegistrationResponse{QueueURL: globalSqsQueue})
	if err != nil {
		common.Fatal("[BROKER] Errore nel marshalling della risposta al publisher. " + err.Error())
		http.Error(w, "Error in response marshalling.\n" + err.Error(), http.StatusInternalServerError)
		return
	}

}

//Aggiorna posizione del subscriber
func handlePositionUpdate(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	positionUpdate := common.SubPositionUpdateRequest{}

	err := json.NewDecoder(r.Body).Decode(&positionUpdate)
	if err != nil {
		common.Fatal("[BROKER] Errore nel unmarshalling della richiesta. " + err.Error())
		http.Error(w, "Error in request marshalling.\n" + err.Error(), http.StatusBadRequest)
		return
	}


	common.Info("[BROKER] Comando di position update del subscriber: " + id + " [ " + positionUpdate.PositionX + ", " + positionUpdate.PositionY + "]")

	err = updatePosition(id, positionUpdate.PositionX, positionUpdate.PositionY)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'aggiornamento della posizione per: " + id + ". " + err.Error())
		http.Error(w, "Error in updating subscriber position.\n" + err.Error(), http.StatusInternalServerError)
		return
	}

}


//Sottoscrizione ai topic di un subscriber
func handleTopicSubscribe(w http.ResponseWriter, r *http.Request) {


	vars := mux.Vars(r)
	id := vars["id"]


	topics := common.SubTopicSubscribeRequest{}

	err := json.NewDecoder(r.Body).Decode(&topics)
	if err != nil {
		common.Fatal("[BROKER] Errore nel unmarshalling della richiesta. " + err.Error())
		http.Error(w, "Error in request marshalling.\n" + err.Error(), http.StatusBadRequest)
		return
	}

	common.Info("[BROKER] Topic da aggiungere al subscriber " + id + ": [ " + common.ConcatenateArrayValues(topics.Topics, ",") + "]")


	err = addTopic(id, topics.Topics)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'aggiunta dei topics. " + err.Error())
		http.Error(w, "Error adding topics.\n" + err.Error(), http.StatusInternalServerError)
		return

	}

}


//Gestisce l'unsubscribe di un topic da parte di un subscriber
func handleTopicUnsubscribe(w http.ResponseWriter, r *http.Request)  {


	vars := mux.Vars(r)
	id := vars["id"]

	topics := common.SubTopicSubscribeRequest{}

	err := json.NewDecoder(r.Body).Decode(&topics)
	if err != nil {
		common.Fatal("[BROKER] Errore nel unmarshalling della richiesta. " + err.Error())
		http.Error(w, "Error in request marshalling.\n" + err.Error(), http.StatusBadRequest)
		return
	}

	common.Info("[BROKER] Topic da rimuovere al subscriber " + id + ": [ " + common.ConcatenateArrayValues(topics.Topics, ",") + "]")


	err = removeTopic(id, topics.Topics)
	if err != nil {
		common.Fatal("[BROKER] Errore nella rimozione dei topics. " + err.Error())
		http.Error(w, "Error removing topics.\n" + err.Error(), http.StatusInternalServerError)
		return

	}
}


//Funzione che rimuove il subscriber dal sistema
func handleSubscriberRemoval(w http.ResponseWriter, r *http.Request) {


	vars := mux.Vars(r)
	id := vars["id"]

	common.Info("[BROKER] Rimozione subscriber " + id)

	err := deleteSubscriber(id)
	if err != nil {
		common.Warning("[BROKER] Errore nella rimozione del subscriber " + id + ". " + err.Error())
		http.Error(w, "Error removing subscriber.\n" + err.Error(), http.StatusInternalServerError)
		return
	}

}






