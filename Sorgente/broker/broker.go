package main
import (
	"common"
	"errors"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/aws/aws-sdk-go/service/sqs"
	"math/rand"
	"strconv"
	"time"
	"net"
	"fmt"
)

/*
			broker.go

	Questo modulo si occupa di gestire la comunicazione tra publisher e subscribers, cioè il broker. Accetta richieste API REST e interagisce
		con la struttura su AWS (code SQS e DynamoDB)

*/


//Punto di ingresso del broker
func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	Init()
}

//Inizializzazione del broker
func Init() {

	//Inizializzazione dell'ambiente
	err := common.InitializeEnvironment()
	if err != nil {
		common.Fatal("[BROKER] Errore nell'inizializzazione dell'applicazione\n" + err.Error())
		return
	}

	//Recupero della configurazione dal dynamoDB
	err = retreiveConfig()
	if err != nil {
		common.Fatal("[BROKER] Errore nel retreive della configurazione\n" + err.Error())
		return
	}
	//Invio messaggio al logger remoto
	sendLogMessage("Configurazione completata")

	//Inizializzo il thread che gestisce le richieste API REST
	go handleRequests()


	broker()

}

// Il meotodo principale del broker che gestisce la logica
func broker() {

	//Go routine per l'aggiornamento della configurazione
	go loadUpdatedConfiguration()

	//Ciclo infinito
	for {
		_, err := receiveQueueMessage(globalSqsQueue)
		if err != nil {
			common.Fatal("[BROKER] Errore nell'ottenimento del messaggio in coda. " + err.Error())
		}

		//Attesa prima di interrogare coda SQS di nuovo
		delay, err := strconv.Atoi(delay_sqs_request)
		if err != nil {
			common.Warning("[BROKER] Errore nella conversione del delay SQS. " + err.Error())
			time.Sleep(time.Second * 10)
		}
		time.Sleep(time.Second * time.Duration(delay))
	}
}


//Metodo per aggiornare automaticamente la configurazione dal DynamoDB relativo
func loadUpdatedConfiguration(){

	for {

		//Tempo di attesa prima di aggiornare la propria configurazione
		delay, err := strconv.Atoi(delay_load_config)
		if err != nil {
			common.Warning("[BROKER] Errore nella conversione del delay per il load della configurazione. " + err.Error())
			time.Sleep(time.Second * 120)
		}
		time.Sleep(time.Second * time.Duration(delay))

		_ = retreiveConfig()
	}
}



//Invio del messaggio alle rispettive code
func sendMessage(message sqs.Message) (retErr error) {

	//Esportazione dei parametri del messaggio sqs.Message ottenuto
	id 				:= *message.MessageAttributes["ID"].StringValue
	topic 			:= *message.MessageAttributes["Topic"].StringValue
	positive, err 	:= strconv.Atoi(*message.MessageAttributes["Positive"].StringValue)
	if err != nil { return errors.New("error converting string to int") }
	peopleNum, err 	:= strconv.Atoi(*message.MessageAttributes["PeopleNum"].StringValue)
	if err != nil { return errors.New("error converting string to int") }
	mq, err 		:= strconv.Atoi(*message.MessageAttributes["Mq"].StringValue)
	if err != nil { return errors.New("error converting string to int") }
	positionX, err 	:= strconv.Atoi(*message.MessageAttributes["PositionX"].StringValue)
	if err != nil { return errors.New("error converting string to int") }
	positionY, err 	:= strconv.Atoi(*message.MessageAttributes["PositionY"].StringValue)
	if err != nil { return errors.New("error converting string to int") }
	radius, err		:= strconv.Atoi(*message.MessageAttributes["Radius"].StringValue)
	if err != nil { return errors.New("error converting string to int") }


	//Filtro per effettuare la query su DynamoDB
	var filter expression.ConditionBuilder

	//Se esiste almeno un caso positivo (messaggio di positività)
	if positive > 0 {
		//Inoltra a tutti a prescindere dai topic, utilizzo il raggio nel caso di positivo
		cond2 := expression.Name("PositionX").Between(expression.Value(positionX - positive_radius), expression.Value(positionX + positive_radius))
		cond3 := expression.Name("PositionY").Between(expression.Value(positionY - positive_radius), expression.Value(positionY + positive_radius))
		filter = expression.And(cond2, cond3)

		//Se il raggio è maggiore di zero (messaggio normale)
	} else if radius > 0 {
		cond1 := expression.Name("Topics").Contains(topic)
		cond2 := expression.Name("PositionX").Between(expression.Value(positionX - radius), expression.Value(positionX + radius))
		cond3 := expression.Name("PositionY").Between(expression.Value(positionY - radius), expression.Value(positionY + radius))
		filter = expression.And(cond1, expression.And(cond2, cond3))

		//Messaggio di emergenza se ha radius == 0 (radius < 0 non è contemplato)
	} else {
		filter = expression.Name("Topics").Contains(topic)
	}

	//Creazione dell'espressione da dare come input
	projection := expression.NamesList(expression.Name("SubID"), expression.Name("Topics"), expression.Name("QueueURL"), expression.Name("PositionX"), expression.Name("PositionY"))
	expr, err := expression.NewBuilder().WithFilter(filter).WithProjection(projection).Build()
	if err != nil {
		common.Fatal("[BROKER] Errore nella costruzione della query. " + err.Error())
	}

	//Esecuzione della query con il filtro
	subsID, queueUrl, err := getFilteredSubscribers(expr)
	if err != nil {
		common.Warning("[BROKER] Errore nell'esecuzione della query a DynamoDB. " + err.Error())
		return err
	}

	//invio del messaggio a tutti i subscriber interessati
	for _, url := range queueUrl {
		err = sendQueueMessage(message, url)
		if err != nil {
			common.Fatal("[BROKER] Errore nell'invio del messaggio. " + err.Error())

		}
	}

	common.Info("[BROKER] Messaggio Ricevuto:\n" +
		"\t[Struttura: " + id + "; Metri quadri: " + strconv.Itoa(mq) + "]: \"" + *message.Body + "\"\n" +
		"\t | Numero persone: " + strconv.Itoa(peopleNum) + " (Positivi: " + strconv.Itoa(positive) + ") \n" +
		"\t | Topic \"" + topic + "\"\n" +
		"\t | Posizione : Raggio (" + strconv.Itoa(positionX) + ", " + strconv.Itoa(positionY) + ") : " + strconv.Itoa(radius) + "\n" +
		"\t | Inoltrato ai subscriber:\n\t |\t | " + common.ConcatenateArrayValues(subsID,"\n\t |\t | ") + "\n" +
		"\t +-----------------------------------------------------------------------------\n")

	sendLogMessage("Messaggio Ricevuto:\n" +
		"\t[Struttura: " + id + "; Metri quadri: " + strconv.Itoa(mq) + "]: \"" + *message.Body + "\"\n" +
		"\t | Numero persone: " + strconv.Itoa(peopleNum) + " (Positivi: " + strconv.Itoa(positive) + ") \n" +
		"\t | Topic \"" + topic + "\"\n" +
		"\t | Posizione : Raggio (" + strconv.Itoa(positionX) + ", " + strconv.Itoa(positionY) + ") : " + strconv.Itoa(radius) + "\n" +
		"\t | Inoltrato ai subscriber:\n\t |\t | " + common.ConcatenateArrayValues(subsID,"\n\t |\t | ") + "\n" +
		"\t +-----------------------------------------------------------------------------\n")

	//Notifica di emergenza nel caso è presente una concetrazione di persone al metro quadro superiore al valore previsto
	if (float64(peopleNum) / float64(mq)) >= mq_threshold {
		sendLogMessage("[ALERT!] Nella struttura " + id + " è stato riscontrata una concentrazione di persone al metro quadro superiore al limite consentito" +
			"\n\t | " + id + ": " + strconv.FormatFloat(float64(peopleNum) / float64(mq), 'f', -1, 64) + " persone/mq, Limite consentito: " + strconv.FormatFloat(mq_threshold, 'f', -1, 64) + " persone/mq\n" +
			"\t +-----------------------------------------------------------------------------\n")
		common.Info("[BROKER] [ALERT!] Nella struttura " + id + " è stato riscontrata una concentrazione di persone al metro quadro superiore al limite consentito" +
			"\n\t | " + id + ": " + strconv.FormatFloat(float64(peopleNum) / float64(mq), 'f', -1, 64) + " persone/mq, Limite consentito: " + strconv.FormatFloat(mq_threshold, 'f', -1, 64) + " persone/mq\n" +
			"\t +-----------------------------------------------------------------------------\n")

	}

	//Se esiste almeno un positivo, invio della notifica di emergenza
	if positive > 0 {
		sendLogMessage("[ALERT!] Nella struttura " + id + " è stato riscontrato un numero di " + strconv.Itoa(positive) + " persone positive.\n")
		common.Info("[BROKER] [ALERT!] Nella struttura " + id + " è stato riscontrato un numero di " + strconv.Itoa(positive) + " persone positive.\n")
	}

	return nil

}


//Funzione per l'invio al logger remoto
func sendLogMessage(message string) {

	prefix := "[BROKER] "

	err := common.SendMessage(common.RemoteLogConnection, prefix + message + "\n")
	if err != nil {
		common.Warning("Errore nel logging remoto")
		
		//Istanzio la connessione con il remote logger
		common.RemoteLogConnection, err = net.Dial("tcp", common.Config.LoggerHost)
		if err != nil {
			fmt.Println("Errore in dial: " + err.Error())
		}
		err = common.SendMessage(common.RemoteLogConnection, "d\n")
		if err != nil {
			fmt.Println("Errore nell'inizializzazione della connessione: " + err.Error())
		}


	common.SendMessage(common.RemoteLogConnection, prefix + message + "\n")

	}

}
