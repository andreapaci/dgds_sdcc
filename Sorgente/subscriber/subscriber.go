package main

import (
	"bufio"
	"common"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"net"
)

/*
			subscriber.go

	Questo è il modulo "start" del subscriber che orchestra il suo funzionamento. Può essere eseguito in modalità "interattiva" e non.
	Come specificato nelle istruzioni, accetta parametri a riga di comando.


*/

//Entry point del subscriber
func main() {

	rand.Seed(time.Now().UTC().UnixNano())

	//Recupero gli argomenti in ingresso (scartando il primo che contiene il nome del programma)
	args := os.Args[1:]

	//Se non si specificano parametri di ingresso, il programma di default sarà interattivo
	interactive := true

	//Se specifica almeno il primo argomento, ed esso è uguale ad "i", allora il programma sarà interattivo
	if len(args) >= 1 {
		if args[0] == "i" { interactive = true
		} else { interactive = false }
	}

	//Se vengono specificati tutti i parametri si parte da una situazione di subscriber predefinita
	if len(args) >= 6 {
		subId := args[1]
		receiveQueue := args[2]
		positionX := args[3]
		positionY := args[4]
		topics := args[5:]

		//Eseguo il subscriber
		run(subId, receiveQueue, topics, positionX, positionY, interactive)

	} else {

		//Se non sono stati specificati tutti i parametri, il publisher viene eseguito impostando valori randomici
		subId := ""
		receiveQueue := ""
		positionX := strconv.Itoa(rand.Intn(common.TestPositionSize))
		positionY := strconv.Itoa(rand.Intn(common.TestPositionSize))

		//Numero randomico di iscrizione ai topics
		var topics []string
		for i := 0; i < rand.Intn(len(common.Topics) - 1) + 1; i++ {
			topics = append(topics, common.Topics[rand.Intn(len(common.Topics))])
		}

		//Eseguo il subscriber
		run(subId, receiveQueue, topics, positionX, positionY, interactive)
	}

}






//Punti di inizio del subscriber
func run(subId string, receiveQueue string, topics []string, positionX string, positionY string, interactive bool){

	//Inizializzazione dell'ambienete
	err := common.InitializeEnvironment()
	if err != nil {
		common.Fatal("[SUB] Errore nella instaurazionde del subscriber\n" + err.Error())
		return
	}

	//Se non è stato fornito un subID o una coda, si esegue la registrazione
	if subId == "" || receiveQueue == "" {

		//Registrazione come subscriber
		for {
			subId, receiveQueue, err = register()
			if err != nil {
				common.Fatal("[SUB] Errore nella registrazione come subscriber. Tentativo di riconnessione tra " + strconv.Itoa(common.Config.RetryDelay) + "s\n" + err.Error())

			} else { break }

			//Ritenta dopo alcuni secondi se non è stato possibile registrarsi
			time.Sleep(time.Second * time.Duration(common.Config.RetryDelay))
		}


	}
	//Invio messaggio al logger remoto
	sendLogMessage(subId, "Configurazione e registrazione completata")

	//Sottoscrizione ai topics
	err = subscribeTopic(subId, topics)
	if err != nil {
		common.Warning("[SUB] Errore nella sottoscrizione ad un topic. " + err.Error())
	}



	//Se interattivo
	if interactive {

		//Se sono stati forniti valori validi per la posizione, vengono registrati, altrimenti viene impostato 0 come valore
		intPositionX, err := strconv.Atoi(positionX)
		if err != nil {
			common.Fatal("[SUB] Errore nella conversione in intero della posizioneX. " + err.Error())
			intPositionX = 0
		}

		intPositionY, err := strconv.Atoi(positionY)
		if err != nil {
			common.Fatal("[SUB] Errore nella conversione int intero della posizioneY. " + err.Error())
			intPositionY = 0
		}
		//Aggiorna la posizione
		err = updateSubscriberPosition(subId, intPositionX, intPositionY)
		if err != nil { common.Warning("[SUB] Errore nell'aggiornamento della posizione. " + err.Error()) }

		//Eseguo il subscriber in modalità interattiva
		interactiveSubscriber(subId, receiveQueue)

	} else {

		//position_update simula la variazione di cordinate GPS nei terminali (specialmente mobili) con una goroutine separata
		go position_update(subId, positionX, positionY)

		//Eseguo il subscriber
		go subscriber(subId, receiveQueue)

		//Dopo Config.SimulationTime secondi, la simulazione termina
		time.Sleep(time.Second * time.Duration(common.Config.SimulationTime))



	}

	//Cleanup dell'ambniente rimuovendo il suibscriber
	err = unsubscribe(subId)
	if err != nil {
		common.Warning("[SUB] Errore nella deregistrazione. " + err.Error())
	}
	common.Info("[SUB] Simulazione terminata")
	return
}

//logica del subscriber (NON INTERATTIVO)
func subscriber(subId string, receiveQueue string){

	//goroutine che emula l'interazione sell'utente
	go handleRandomUserInteraction(subId)

	//Loop per la ricezione dei messaggi
	for {

		_, err  := receiveQueueMessage(subId, receiveQueue)
		if err != nil  {
			common.Fatal("[SUB] Errore nella ricezione del messaggio. " + err.Error())
		}

		time.Sleep(time.Second * time.Duration(common.Config.RcvMessDelay))
	}

}





// >Metodo che emula interazione dell'utente con un comportamento randomico nel caso di una esecuzione non interattiva
func handleRandomUserInteraction(subId string){

	//Aggiunto un ritardo per simulare una interazione dell'utente a tempo discreto
	time.Sleep(time.Second * time.Duration(common.Config.OpDelay))

	//La variabile randomica "choice" emula un evento randomico che l'applicativo riceve. L'applicazione infatti si pone semplicemente come
	// GATEWAY tra l'eventuale interfaccia utente del subscriber (es. app per smartphone) e il resto del sistema (BROKER).
	//Data l'assenza di un ambiente reale per l'applicazione in cui girare, questi eventi vengono "mockati", cioè simulati.
	choice := rand.Float64()

	//Sottoscrizione ad un topic
	if choice < 0.1 {
		err := subscribeTopic(subId, []string{ common.Topics[rand.Intn(len(common.Topics))] })
		if err != nil {
			common.Warning("[SUB] Errore nella rimozione di un topic. " + err.Error())
		}

	//Deregistrazione ad un topic
	} else if choice < 0.2 {

		err := unsubscribeTopic(subId, []string{ common.Topics[rand.Intn(len(common.Topics))] })
		if err != nil {
			common.Warning("[SUB] Errore nella rimozione di un topic. " + err.Error())
		}
	}


}





//Funzione per il subscriber interattivo. Questo permette di simulare un comportamento specifico dell'applicazione per testare le sue funzionalità
// Nota: essendo un'applicativo per testare l'invio di messaggi, e quindi non facente veramente parte dell'infrastruttura, non è stato posta particolare
//attenzione sulla validazione dell'input da linea di comando
func interactiveSubscriber(subId string, receiveQueue string){

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n\nQuando si vede il tag [INPUT] è necessario l'input dell'utente.")

	for{

		fmt.Print("\n[INPUT] Inserire il numero dell'operazione che si intende fare:" +
			"\n\t - 1: Ricevere messaggio dalla coda" +
			"\n\t - 2: Iscriversi ad uno o più topic" +
			"\n\t - 3: Disiscriversi ad uno o più topic" +
			"\n\t - 4: Aggiornare la posizione del subscriber" +
			"\n\t - Qualsiasi carattere: Disiscriversi ed uscire\nInput: ")

		input, err := common.ReadInput(reader)
		if err != nil { return }


		//Ricezione messaggio dalla coda
		if strings.Compare(input, "1") == 0 {

			fmt.Print("Ricezione del messaggio come subscriber " + subId + " :" +
				"(Si potrebbe attendere fino a " + strconv.FormatInt(common.Config.PollingTime, 10) + " secondi per il Polling time\n . . .\n")

			_, err  := receiveQueueMessage(subId, receiveQueue)
			if err != nil  {
				common.Fatal("[SUB] Errore nella ricezione del messaggio. " + err.Error())
			}

		//Sottoscrizione ai topics
		} else if strings.Compare(input, "2") == 0 {

			fmt.Print("Topic disponibili (e' possibile usare valori al di fuori di quelli proposti): \n\t - " + common.ConcatenateArrayValues(common.Topics, "\n\t - ") + "\n")

			fmt.Print("[INPUT] Inserire i topics a cui ci si vuole iscrivere, separati da uno spazio (es. Ristorazione Farmacia ): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }


			//Ottenimento di tutti i topic separati da " "
			topics := strings.Split(input, " ")

			err := subscribeTopic(subId, topics)
			if err != nil {
				common.Fatal("[SUB] Errore nell'iscrizione di topic. " + err.Error())
			}

		//Deregistrazione ai topic
		} else if strings.Compare(input, "3") == 0 {

			fmt.Print("Topic disponibili (e' possibile usare valori al di fuori di quelli proposti): \n\t - " + common.ConcatenateArrayValues(common.Topics, "\n\t - ") + "\n")

			fmt.Print("[INPUT] Inserire i topics a cui ci si vuole disiscrivere, separati da uno spazio (es. Ristorazione Farmacia ): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }


			topics := strings.Split(input, " ")

			err := unsubscribeTopic(subId, topics)
			if err != nil {
				common.Fatal("[SUB] Errore nella disiscrizione di topic. " + err.Error())
			}


		//Aggiornamento della posizione
		} else if strings.Compare(input, "4") == 0 {


			fmt.Print("[INPUT*] Specificare la coordinata X [numero intero positivo] del subscriber: ")

			input, err = common.ReadInput(reader)
			if err != nil { return }

			positionX, err := strconv.Atoi(input)
			if err != nil { common.Fatal("[SUB] Errore nell'input immesso. " + err.Error()) }




			fmt.Print("[INPUT*] Specificare la coordinata Y [numero intero positivo] del subscriber: ")

			input, err = common.ReadInput(reader)
			if err != nil { return }

			positionY, err := strconv.Atoi(input)
			if err != nil { common.Fatal("[SUB] Errore nell'input immesso. " + err.Error()) }


			err = updateSubscriberPosition(subId, positionX, positionY)
			if err != nil {
				common.Warning("[SUB] Errore nell'aggiornamento della posizione. " + err.Error())
			}

		} else { return }

	}

}






//Ricezione del messaggio in coda
func receiveQueueMessage(subid string, receiveQueue string) (messages []*sqs.Message, retErr error){


	svc := sqs.New(common.Sess)
	var messagesList []*sqs.Message

	result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		WaitTimeSeconds: aws.Int64(common.Config.PollingTime),
		MaxNumberOfMessages: aws.Int64(common.Config.MaxRcvMessage),
		QueueUrl: &receiveQueue,

	})
	if err != nil {
		common.Warning("[SUB] Errore nell'ottenimento del messaggio. " + err.Error())
		return nil, err
	}
	if len(result.Messages) == 0 {
		common.Info("[SUB] Nessun messaggio ricevuto")
		return
	} else {

		sendLogMessage(subid, "Messaggi ricevuti: " + strconv.Itoa(len(result.Messages)))
		common.Info("[SUB " + subid +"] Messaggi ricevuti: " + strconv.Itoa(len(result.Messages)))

		//Ciclo per tutti i messaggi ricevuti
		for _, mess := range result.Messages {

			_, err := svc.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      &receiveQueue,
				ReceiptHandle: mess.ReceiptHandle,
			})
			if err != nil {
				common.Info("[SUB] Errore nell'eliminazione del messaggio. " + err.Error())

				//Solo se ottengo il messaggio con successo lo stampo (per ottenerlo con successo vuol dire che sono riuscito ad eliminarlo
			} else {
				messagesList = append(messagesList, mess)

				id 				:= *mess.MessageAttributes["ID"].StringValue
				topic 			:= *mess.MessageAttributes["Topic"].StringValue
				positive		:= *mess.MessageAttributes["Positive"].StringValue
				peopleNum	  	:= *mess.MessageAttributes["PeopleNum"].StringValue
				mq		 		:= *mess.MessageAttributes["Mq"].StringValue


				common.Info("[SUB] Messaggio Ricevuto:\n" +
					"\t[Struttura: " + id + "; Metri quadri: " + mq + "]: \"" + *mess.Body + "\"\n" +
					"\t | Numero persone: " + peopleNum + " (Positivi: " + positive + ") \n" +
					"\t | Topic \"" + topic + "\"\n" +
					"\t +-----------------------------------------------------------------------------\n")

				sendLogMessage(subid, "Messaggio Ricevuto:\n" +
					"\t[Struttura: " + id + "; Metri quadri: " + mq + "]: \"" + *mess.Body + "\"\n" +
					"\t | Numero persone: " + peopleNum + " (Positivi: " + positive + ") \n" +
					"\t | Topic \"" + topic + "\"\n" +
					"\t +-----------------------------------------------------------------------------\n")

				common.Info("[SUB] Messaggio eliminato con successo")
			}
		}

	}



	return messagesList, nil

}



//Invia messaggio al logger remoto
func sendLogMessage(id string, message string){

	prefix := "[SUB " + id + "] "

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




