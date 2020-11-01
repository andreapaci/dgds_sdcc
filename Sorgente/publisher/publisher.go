package main

import (
	"bufio"
	"common"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"net"
)

/*
			publisher.go

	Questo è il modulo "start" del publisher che orchestra il suo funzionamento. Può essere eseguito in modalità "interattiva" e non.
	Come specificato nelle istruzioni, accetta parametri a riga di comando.

*/


//Variabile che contiene l'URL della coda SQS per i messaggi in uscita
var sendQueue string

// Punto di ingresso
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

	//Se vengono specificati tutti i parametri si parte da una situazione di publisher predefinita
	if len(args) >= 8 {

		name := args[1]
		topic := args[2]
		peopleNum := args[3]
		positionX := args[4]
		positionY := args[5]
		radius := args[6]
		mq := args[7]

		//Eseguo il publsiher
		run(name, topic, peopleNum, positionX, positionY, radius, mq, interactive)

	} else {
		//Se non sono stati specificati tutti i parametri, il publisher viene eseguito impostando valori randomici

		name := "Struttura #" + strconv.Itoa(rand.Intn(100))				//Nome struttura
		topic := common.Topics[rand.Intn(len(common.Topics))]				//Topic a cui manderà i messaggi
		peopleNum := strconv.Itoa(rand.Intn(50) + 20)					//Numero di persone presenti nella struttura al momento dello startup
		positionX := strconv.Itoa(rand.Intn(common.TestPositionSize))		//Posizione della struttura in coordinate X (Nota: Le cordinate X, Y si riferiscono al "blocco" posizionato in X,Y, non sono latitudine e longitudine)
		positionY := strconv.Itoa(rand.Intn(common.TestPositionSize))		//Posizione della struttura in coordinate Y
		radius := strconv.Itoa(rand.Intn(10) + 1)						//Raggio di interesse per il messaggio mandato dal publisher (Distanza per la quale i subscriber riceveranno il messaggio del publisher)
		mq := strconv.Itoa(rand.Intn(100))								//Metri quadri della struttura

		//Eseguo il publsiher
		run(name, topic, peopleNum, positionX, positionY, radius, mq, interactive)
	}

}




// Punto di inizio del publisher
func run(name string, topic string, peopleNum string, positionX string, positionY string, radius string, mq string, interactive bool) {

	//Inizializzazione dell'ambienete
	err := common.InitializeEnvironment()
	if err != nil {
		common.Fatal("[PUB] Errore nella instaurazione del publisher\n" + err.Error())
		return
	}


	time.Sleep(time.Second * 1)

	//Comunicazione con il broker per ottenere la coda SQS su cui mandare il messaggio
	for {
		err = getSendQueue()
		if err != nil {
			common.Fatal("[PUB] Errore nell'ottenimento della coda dal broker. Tentativo di riconnessione tra " + strconv.Itoa(common.Config.RetryDelay) + "s\n" + err.Error())
			time.Sleep(time.Second * time.Duration(common.Config.RetryDelay))	//Se connessione con broker fallisce, si ritenta dopo "Config.RetryDelay" secondi.
		} else { break }
	}

	//Invio del messaggio al log remoto
	sendLogMessage("Configurazione completata")

	if interactive {
		//Eseguo in maniera interattiva il publisher
		interactivePublisher(name, topic, peopleNum, positionX, positionY, radius, mq)

	} else {
		//Applicazione non interattiva
		go publisher(name, topic, peopleNum, positionX, positionY, radius, mq)

		time.Sleep(time.Second * time.Duration(common.Config.SimulationTime)) 	//La simulazione terminerà dopo Config.SimulationTime secondi.
	}

	//Invio del messaggio al log remoto
	sendLogMessage("Simulazione terminata")
	common.Info("[PUB] Simulazione terminata")
}




// Logica del publisher (NON INTERATTIVO)
func publisher(name string, topic string, peopleNum string, positionX string, positionY string, radius string, mq string) {

	delay := time.Duration(common.Config.OpDelay)


	for {

		//La variabile randomica "choice" emula un evento randomico che l'applicativo riceve. L'applicazione infatti si pone semplicemente come
		// GATEWAY tra sensori di varia natura (come per esempio un modulo bluetooth per il conteggio delle persone) e il resto del sistema (BROKER).
		//Data l'assenza di un ambiente reale per l'applicazione in cui girare, questi eventi vengono "mockati", cioè simulati.
		choice := rand.Float64()

		//Evento nuovi positivi trovati nella struttura
		if choice < 0.1 {

			positive := strconv.Itoa(rand.Intn(4) + 1)

			err := sendQueueMessage("Numeno nuovi positivi dall'ultima segnalazione: " + positive, name, peopleNum, positive, mq, topic, positionX, positionY, radius)
			if err != nil {
				common.Warning(err.Error())
			}

		//Segnalazione generale da inoltrare a tutti i subscriber sottoscritti al Topic utilizzato (può essere per esempio usato per mandare comunicazioni di servizio di carattere generale)
		} else if choice < 0.3 {


			err := sendQueueMessage("Segnalazione di emergenza (inoltrato a tutti i subscriber del topic \"" + topic + "\")", name, peopleNum, "0", mq, topic, positionX, positionY, "0")
			if err != nil {
				common.Warning(err.Error())
			}

		//Variazione del numero di persone nella struttura
		} else if choice < 0.7 {

			intPeopleNum, err := strconv.Atoi(peopleNum)
			if err != nil {
				common.Fatal("[PUB] Errore nella conversione del numero di persone. " + err.Error())
			} else {
				// (rand.Intn(3) - 1) esprime un valore random tra -1, 0, 1; che moltiplicato per un numero tra 0 e 4 esprime la variazione di persone
				peopleNum = strconv.Itoa(intPeopleNum  + ((rand.Intn(3) - 1) * rand.Intn(10))) // (rand.Intn(3) - 1) esprime un valore random tra -1, 0, 1; che moltiplicato per un numero tra 0 e 4 esprime la variazione di persone
			}

			err = sendQueueMessage("Numero di persone presenti attualmente nella struttura: " + peopleNum, name, peopleNum, "0", mq, topic, positionX, positionY, radius)
			if err != nil {
				common.Warning(err.Error())
			}


		//Simulazione di un numero di persone concentrate più del dovuto (rapporto di persone/metro_quadro > soglia)
		} else if choice < 0.9 {

			fakePeopleNum := ""

			intMq, err := strconv.Atoi(mq)
			if err != nil {
				common.Fatal("[PUB] Errore nella conversione dei metri quadri della struttura. " + err.Error())
			} else {
				fakePeopleNum = strconv.Itoa(2 * intMq)
			}

			err = sendQueueMessage("Numero di persone presenti attualmente nella struttura: " + fakePeopleNum, name, fakePeopleNum, "0", mq, topic, positionX, positionY, radius)
			if err != nil {
				common.Warning(err.Error())
			}

		}

		//Ritardo tra un'operazione ed un'altra
		time.Sleep(time.Second * delay)
	}
}





//Funzione per il publisher interattivo. Questo permette di simulare un comportamento specifico dell'applicazione per testare le sue funzionalità
// Nota: essendo un'applicativo per testare l'invio di messaggi, e quindi non facente veramente parte dell'infrastruttura, non è stato posta particolare
//attenzione sulla validazione dell'input da linea di comando
func interactivePublisher(name string, topic string, peopleNum string, positionX string, positionY string, radius string, mq string){

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n\nQuando si vede il tag [INPUT] è necessario l'input dell'utente.")

	for{

		fmt.Print("\n[INPUT] Inserire C per inviare un messaggio o un qualsiasi altro carattere per uscire: ")

		input, err := common.ReadInput(reader)
		if err != nil { return }

		if strings.Compare(strings.ToLower(input), "c") == 0{

			fmt.Println("Le richieste marchate con * nel tag [INPUT] (es. [INPUT*]) possono essere saltata premendo ENTER per utilizzare il valore già impostato.\n")



			//---------- Nome struttura ----------
			fmt.Print("[INPUT*] Quale nome (identificativo) della struttura si intende usare (valore impostato = " + name + "): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }

			if input != "" { name = input }



			//---------- Topic da utilizzare ----------
			fmt.Print("Topic disponibili (e' possibile usare valori al di fuori di quelli proposti): \n\t - " + common.ConcatenateArrayValues(common.Topics, "\n\t - ") + "\n")
			fmt.Print("[INPUT*] A quale topic si vuole mandare il messaggio (valore impostato = " + topic + "): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }

			if strings.Compare(input, "") != 0 { topic = input }




			//---------- Coordinata X e Y ----------
			fmt.Print("[INPUT*] Specificare la coordinata X [numero intero positivo] del publisher (valore impostato = " + positionX + ")\n" +
				"(E' consigliato utilizzare un valore al di sotto di " + strconv.Itoa(common.TestPositionSize) + " ai fini del test): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }

			if strings.Compare(input, "") != 0 { positionX = input }



			fmt.Print("[INPUT*] Specificare la coordinata Y [numero intero positivo] del publisher (valore impostato = " + positionY + ")\n" +
				"(E' consigliato utilizzare un valore al di sotto di " + strconv.Itoa(common.TestPositionSize) + " ai fini del test): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }

			if strings.Compare(input, "") != 0 { positionY = input }




			//---------- Raggio publicazione ----------
			fmt.Print("[INPUT*] Specificare il raggio della publicazione [numero intero positivo] del publisher (valore impostato = " + radius + ")\n" +
				"(Il raggio è il numero di \"blocchi\" e non il raggio in metri effettivo)\n" +
				"(Con il valore 0 il messaggio viene inviato a tutti i subscriber registrati al topic utilizzato, a prescindere dalla posizione): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }

			if strings.Compare(input, "") != 0 { radius = input }



			//---------- Metri quadri ----------
			fmt.Print("[INPUT*] Specificare i metri quadri della struttura [numero intero positivo] del publisher (valore impostato = " + mq + "): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }

			if strings.Compare(input, "") != 0 { mq = input }



			//---------- Numero persone nella struttura ----------
			fmt.Print("[INPUT*] Specificare il numero di persone nella struttura [numero intero positivo] del publisher (valore impostato = " + peopleNum + "): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }

			if strings.Compare(input, "") != 0 { peopleNum = input }



			//---------- Nuovi positivi ----------
			fmt.Print("[INPUT*] Specificare il numero di persone (nuove) positive trovate nella struttura dall'ultima segnalazione (valore impostato = 0): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }
			if input == "" { input = "0" }
			positive := input



			//---------- Testo del messaggio ----------
			fmt.Print("[INPUT*] Inserire il testo del messaggio (valore impostato = Nessun messaggio): ")

			input, err = common.ReadInput(reader)
			if err != nil { return }
			if input == "" { input = "Nessun messaggio"}
			message := input


			//Invio del messaggio
			if sendQueueMessage(message, name, peopleNum, positive, mq, topic, positionX, positionY, radius) != nil {
				common.Fatal("[PUB] Errore nell'invio del messaggio. " + err.Error())
			}


		} else { return }


	}
}




//Funzione che invia il messaggio alla coda SQS verso il broker
func sendQueueMessage(message string, id string, peopleNum string, positive string, mq string, topic string, positionX string, positionY string, radius string) (retErr error) {

	rad, _ := strconv.Atoi(radius)

	//Raggio < 0 non valido
	if rad < 0 {
		common.Warning("[PUB] Raggio di posizione minore di zero")
		return errors.New("radius must be a positive value")
	}

	//Connessione con AWS
	svc := sqs.New(common.Sess)

	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	deduplication_ID := reg.ReplaceAllString(id, "")

	//Creazione e invio del messaggio messaggio
	_, err = svc.SendMessage(&sqs.SendMessageInput{
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"ID": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(id),
			},
			"Positive": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(positive),
			},
			"PeopleNum": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(peopleNum),
			},
			"Mq": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(mq),
			},
			"Topic": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(topic),
			},
			"PositionX": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String(positionX),
			},
			"PositionY": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String(positionY),
			},
			"Radius": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String(radius),
			},
		},
		MessageGroupId:         aws.String(deduplication_ID + "groupID"),
		MessageDeduplicationId: aws.String(deduplication_ID + strconv.Itoa(time.Now().Nanosecond())),
		MessageBody:            aws.String(message),
		QueueUrl:               &sendQueue,
	})
	if err != nil {
		common.Warning("[PUB] Errore nell'invio del messaggio. " + err.Error())
		return err
	}



	sendLogMessage("Messaggio inviato:\n" +
		"\t[Struttura: " + id + "; Metri quadri: " + mq + "]: \"" + message + "\"\n" +
		"\t | Numero persone: " + peopleNum + " (Positivi: " + positive + ") \n" +
		"\t | Topic \"" + topic + "\"\n" +
		"\t | Posizione:Raggio (" + positionX + ", " + positionY + "):" + radius + "\n" +
		"\t +-----------------------------------------------------------------------------\n")

	common.Info("[PUB] Messaggio inviato:\n" +
		"\t[Struttura: " + id + "; Metri quadri: " + mq + "]: \"" + message + "\"\n" +
		"\t | Numero persone: " + peopleNum + " (Positivi: " + positive + ") \n" +
		"\t | Topic \"" + topic + "\"\n" +
		"\t | Posizione : Raggio (" + positionX + ", " + positionY + ") : " + radius + "\n" +
		"\t +-----------------------------------------------------------------------------\n")
	return nil

}


//Interazione iniziale con il broker per ottenere la coda SQS dove mandare i messaggi
func getSendQueue() (retErr error) {

	//Invio richiesta GET
	statusCode, getResponse, err := common.GetRequest(common.Config.AwsBroker + "/publisher", common.PubRegistrationResponse{})
	if err != nil {
		common.Fatal("[PUB] Errore nella registrazione ( " + strconv.Itoa(statusCode) + " ). " + err.Error())
		return err
	}

	//Interpretazione della risposta
	response := common.PubRegistrationResponse{}
	_ = common.FillStruct(&response, getResponse.(map[string]interface{}))

	sendQueue = response.QueueURL

	common.Info("[PUB] Publisher registrato correttamente ( " + strconv.Itoa(statusCode) + " ). Coda: " + sendQueue)

	return nil

}

//Invio messaggio al logger remoto
func sendLogMessage(message string) {

	prefix := "[PUB] "

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
