package main

import (
	"common"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/aws/aws-sdk-go/service/sqs"
	"log"
	"regexp"
	"strconv"
	"time"
)

/*
			broker-aws-interaction.go

	Questo modulo si occupa dell'interazione con AWS da parte del broker, fornendo quelle utility di base come
		scrittura/lettura del Database, creazione/rimozione/consultazione di una coda SQS.
	Queste routine sono state raccolte qui per una maggiore pulizia del codice.

*/


// La funzione esegue una query sul sever DynamoDB per la configurazione
func makeConfigQuery() (configs []ConfigEntry, retErr error) {

	// Creazione DynamoDB client
	svc := dynamodb.New(common.Sess)

	// Determino l'input della query
	input := &dynamodb.ScanInput{
		TableName: aws.String(configTable),
	}

	// Effettuo la query
	result, err := svc.Scan(input)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'esecuzione della Query\n" + err.Error())
		return nil, err
	}

	// Salvo i valori
	index := 0
	common.Info("[BROKER] Query eseguita con successo")

	for _, i := range result.Items {

		config := ConfigEntry{}

		// Unmarshaling del dato ottenuto
		err = dynamodbattribute.UnmarshalMap(i, &config)
		if err != nil {
			common.Fatal("[BROKER] Errore nell'unmarshaling della entry\n" + err.Error())
			return nil, err
		}

		configs = append(configs, config)

		index++
	}

	return configs, nil
}


//Aggiornamento di un parametro di configurazione
func updateConfigurationParameter(fieldName string, fieldValue string) (retErr error) {

	svc := dynamodb.New(common.Sess)

	//Espressione condizionale che impone l'esistenza del parametro (Senza questa condizione, DynamoDB può creare una entry se non trova la corrispettiva chiave nel database)
	cond := "attribute_exists(FieldName)"

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": {
				S: aws.String(fieldValue),
			},
		},
		TableName: aws.String(configTable),
		Key: map[string]*dynamodb.AttributeValue{
			"FieldName": {
				S: aws.String(fieldName),
			},
		},
		ConditionExpression: &cond,
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set FieldValue = :v"),
	}

	//Esecuzione della query
	_, err := svc.UpdateItem(input)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'aggiornamento del parametro di configurazione. " + err.Error())
		return err
	}

	common.Info("[BROKER] Parametro aggiornato")

	return nil
}






//Ottiene la lista di tutti i subscribers registrati nel sistema
func getSubscribers() (subs []common.SubscriberEntry, retErr error) {

	var subList []common.SubscriberEntry

	// Creazione DynamoDB client
	svc := dynamodb.New(common.Sess)

	// Determino l'input della query
	input := &dynamodb.ScanInput{
		TableName: aws.String(subTableName),
	}

	// Effettuo la query
	result, err := svc.Scan(input)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'esecuzione della Query\n" + err.Error())
		return nil, err
	}

	for _, r := range result.Items {

		var subID = common.SubscriberEntry{}

		// Unmarshaling del dato ottenuto
		err = dynamodbattribute.UnmarshalMap(r, &subID)
		if err != nil {
			common.Fatal("Errore nell'unmarshaling della entry\n" + err.Error())
			return nil, err
		}

		subList = append(subList, subID)

	}


	return subList, nil

}

//Ottieni esclusivamente gli ID dei subscribers registrati nel sistema
func getSubscribersID() (subsID []string, retErr error) {

	var subList []string

	// Creazione DynamoDB client
	svc := dynamodb.New(common.Sess)

	// Determino l'input della query
	input := &dynamodb.ScanInput{
		TableName: aws.String(subTableName),
	}

	// Effettuo la query
	result, err := svc.Scan(input)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'esecuzione della Query\n" + err.Error())
		return nil, err
	}

	for _, r := range result.Items {

		var subID = common.SubscriberEntry{}

		// Unmarshaling del dato ottenuto
		err = dynamodbattribute.UnmarshalMap(r, &subID)
		if err != nil {
			common.Fatal("Errore nell'unmarshaling della entry\n" + err.Error())
			return nil, err
		}

		subList = append(subList, subID.SubID)

	}


	return subList, nil

}




//Metodo per aggiungere un subscriber alla tabella su dynamoDB
func addEntryDB(subID string, queueUrl string) (retErr error, alreadyExisting bool) {

	svc := dynamodb.New(common.Sess)

	topics := []string{"empty"}

	//Item da aggiungere
	item := common.SubscriberEntry{
		SubID:     	subID,
		QueueURL:  	queueUrl,
		Topics:    	topics,
		PositionX: 	0,
		PositionY: 	0,
	}

	//Marshalling
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		common.Fatal("[BROKER] Errore nel marshalling della struttura dati")
		return err, false
	}

	//Creazione della query
	cond := "attribute_not_exists(SubID)" 	//Questa condizione è necessaria poiche una ADD su DynamoDB, se trova un elementro con la stessa chiave, esegue un UPDATE invece di annullare la transazione
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(subTableName),
		ConditionExpression: &cond,

	}

	//Esecuzione della query
	_, err = svc.PutItem(input)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'inserimento del subscriber\n" + err.Error())

		switch err.(type) {
		default:
			return err, false
		//L'errore presentato in questa riga viene gestito diversamente, infatti con ogni probabilità indica la registrazione contmeporanea di due subscriber. Questo viene gestito rieffettuando una nuova registrazione
		case *dynamodb.ConditionalCheckFailedException:
			return err, true
		}


	}

	common.Info("[BROKER] Subscriber registrato al DB con successo. (" + item.SubID + ")")

	return nil, false
}




//Aggiornamento della posizione di un subscriber
func updatePosition(subID string, strpositionX string, strpositionY string) (retErr error) {


	svc := dynamodb.New(common.Sess)

	//L'UPDATE in DynamoDB si comporta come un ADD nel momento in cui non trova la chiave. Con questa condizione si previene questo fenomeno
	cond := "attribute_exists(SubID)"

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":x": {
				N: aws.String(strpositionX),
			},
			":y": {
				N: aws.String(strpositionY),
			},
		},
		TableName: aws.String(subTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"SubID": {
				S: aws.String(subID),
			},
		},
		ConditionExpression: &cond,
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set PositionX = :x, PositionY = :y "),
	}

	_, err := svc.UpdateItem(input)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'aggiornamento della posizione. " + err.Error())
		return err
	}

	common.Info("[BROKER] Posizione aggiornata con successo")

	return nil
}


//Aggiunta di 1 o più topic ad un subscriber
func addTopic(subID string, topics []string) (retErr error) {

	//Creazione di un nuovo array da caricare su DynamoDB
	origTopics, err := getTopicList(subID)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'ottenere la topic list")
		return err
	}

	for _, elem := range topics {
		if common.StringListContains(origTopics, elem) == false {
			origTopics = append(origTopics, elem)
		}
	}



	svc := dynamodb.New(common.Sess)

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":t": {
				SS: aws.StringSlice(origTopics),
			},
		},
		TableName: aws.String(subTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"SubID": {
				S: aws.String(subID),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set Topics = :t"),
	}

	_, err = svc.UpdateItem(input)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'aggiunta di topic. " + err.Error())
		return err
	}

	common.Info("[BROKER] Topic aggiunti con successo")

	//Un lista vuota di topic è espressa dal singolo elementro "empty", che viene protnamente eliminato se si aggiungono topics
	_ = removeTopic(subID, []string{"empty"})

	return nil
}





//Rimozione di topic ad un subscriber
func removeTopic(subID string, topics []string) (retErr error) {

	//Creazione nuovo insieme di topic eliminando quelli da rimuovere
	origTopics, err := getTopicList(subID)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'ottenere la topic list")
	}

	var newTopics []string

	for _, elem := range origTopics {
		if common.StringListContains(topics, elem) == false {
			newTopics = append(newTopics, elem)
		}
	}

	//Se non rimane nessun topic, viene impostato il singolo topic empty
	//Nota bene: questo è necessario farlo poichè dynamodb non supporta le liste vuote
	if len(newTopics) == 0 {
		newTopics = append(newTopics, "empty")
	}

	svc := dynamodb.New(common.Sess)

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":t": {
				SS: aws.StringSlice(newTopics),
			},
		},
		TableName: aws.String(subTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"SubID": {
				S: aws.String(subID),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set Topics = :t"),
	}

	_, err = svc.UpdateItem(input)
	if err != nil {
		common.Fatal("[BROKER] Errore nella rimozione di topic. " + err.Error())
		return err
	}

	common.Info("[BROKER] Topic rimosso con successo")

	return nil
}


//Ottiene la lista dei topic di un subscriber
func getTopicList(id string) (topics []string, retErr error) {

	//Creazione client DynamoDB
	svc := dynamodb.New(common.Sess)

	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(subTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"SubID": {
				S: aws.String(id),
			},
		},
	})
	if err != nil {
		common.Warning("[BROKER] Errore nel retreive dell'item con ID: " + id + ".\n" + err.Error())
		return nil, err
	}

	item := common.SubscriberEntry{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		common.Warning("[BROKER] Errore nell'unmarshaling del risultato")
		return nil, err
	}
	if item.SubID == "" {
		common.Warning("[BROKER] Nessun subscriber trovato con id " + id)
		return nil, errors.New("no item found")
	}



	return item.Topics, nil
}


//Ottiene la coda SQS di un subscriber
func getQueueUrl(id string) (queueUrl string, retErr error) {

	//Creazione client DynamoDB
	svc := dynamodb.New(common.Sess)

	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(subTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"SubID": {
				S: aws.String(id),
			},
		},
	})
	if err != nil {
		common.Warning("[BROKER] Errore nel retreive del subscriber con ID: " + id + ".\n" + err.Error())
		return "", err
	}

	item := common.SubscriberEntry{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		common.Warning("[BROKER] Errore nell'unmarshaling del risultato")
		return "", err
	}
	if item.SubID == "" {
		common.Warning("[BROKER] Nessun subscriber trovato con id " + id)
		return "", errors.New("no item found")
	}

	common.Info("[BROKER] Subscriber trovato: " + item.SubID + "\n\t" + item.QueueURL)

	return item.QueueURL, nil
}


//Ottieni i subscriber filtrati con la relativa espressione (Su base topic e context-aware)
func getFilteredSubscribers(expr expression.Expression) (subsID []string, queueUrl []string, retErr error) {

	var subs []string
	var qUrl []string

	svc := dynamodb.New(common.Sess)

	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(subTableName),
	}


	result, err := svc.Scan(params)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'esecuzione della query. " + err.Error())
		return nil, nil, err
	}

	for _, i := range result.Items {
		item := common.SubscriberEntry{}

		err = dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			common.Fatal("[BROKER] Errore nell'unmarshalling della entry. " + err.Error())
			return nil, nil, err
		}

		subs = append(subs, item.SubID)
		qUrl = append(qUrl, item.QueueURL)

	}

	return subs, qUrl, nil

}



//Funzione che rimuove un subscriber dal database
func removeEntryDB(subID string) (retErr error) {

	svc := dynamodb.New(common.Sess)

	//Item da eliminare
	item := common.SubscriberEntry{
		SubID: subID,
	}

	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"SubID": {
				S: aws.String(item.SubID),
			},
		},
		TableName: aws.String(subTableName),
	}

	//Eliminazione dell'item
	_, err := svc.DeleteItem(input)
	if err != nil {
		common.Fatal("[BROKER] Errore nell'eliminazione dell'item\n" + err.Error())
		return err
	}

	common.Info("[BROKER] Subscriber rimosso con successo. (" + item.SubID + ")")

	return nil
}


//Funzione per creare coda SQS per il subID
func createQueue(subID string) (queueUrl string, retErr error) {

	//Creo un service client SQS
	svc := sqs.New(common.Sess)

	//Creo la coda
	result, err := svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(subID + ".fifo"),
		Attributes: map[string]*string{
			"MessageRetentionPeriod":        	aws.String("345600"),
			"ReceiveMessageWaitTimeSeconds": 	aws.String(strconv.FormatInt(common.Config.PollingTime, 10)),
			"FifoQueue":						aws.String("true"),		//Coda FIFO
		},
	})
	if err != nil {
		common.Warning("[BROKER] Errore nella creazione della coda\n" + err.Error())
		return "", err
	}

	common.Info("[BROKER] Coda creata con successo all'URL " + *result.QueueUrl)

	return *result.QueueUrl, nil
}


//Ricezione messaggio sulla coda SQS
func receiveQueueMessage(receiveQueue string) (messages []*sqs.Message, retErr error) {

	svc := sqs.New(common.Sess)
	var messagesList []*sqs.Message

	result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		WaitTimeSeconds:     aws.Int64(common.Config.PollingTime),  	//Long polling
		MaxNumberOfMessages: aws.Int64(common.Config.MaxRcvMessage),
		QueueUrl:            &receiveQueue,
	})


	if err != nil {
		common.Warning("[BROKER] Errore nell'ottenimento del messaggio. " + err.Error())
		return nil, err
	}
	if len(result.Messages) == 0 {
		common.Info("[BROKER] Nessun messaggio ricevuto")
		return
	} else {
		sendLogMessage("Messaggi ricevuti: " + strconv.Itoa(len(result.Messages)))

		for _, mess := range result.Messages {

			err = sendMessage(*mess)

			if err != nil {
				common.Warning("[BROKER] Errore nell'invio del messaggio dal broker. " + err.Error())
			}

			//Messaggio eliminato solo dopoche viene mandato
			_, err := svc.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      &receiveQueue,
				ReceiptHandle: mess.ReceiptHandle,
			})
			if err != nil {
				common.Info("[BROKER] Errore nell'eliminazione del messaggio. " + err.Error())
			} else {
				messagesList = append(messagesList, mess)
				common.Info("[BROKER] Messaggio eliminato con successo")
			}

		}

	}

	return messagesList, nil

}


//Invio del messaggio alla relativa coda
func sendQueueMessage(message sqs.Message, queueUrl string) (retErr error) {

	id 				:= *message.MessageAttributes["ID"].StringValue
	topic 			:= *message.MessageAttributes["Topic"].StringValue
	positive		:= *message.MessageAttributes["Positive"].StringValue
	peopleNum 		:= *message.MessageAttributes["PeopleNum"].StringValue
	mq		 		:= *message.MessageAttributes["Mq"].StringValue

	svc := sqs.New(common.Sess)

	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}

	//Utilizzato per la coda FIFO
	deduplication_ID := reg.ReplaceAllString(queueUrl, "")

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
		},
		MessageGroupId:         aws.String( deduplication_ID + "groupID"),
		MessageDeduplicationId: aws.String(deduplication_ID + strconv.Itoa(time.Now().Nanosecond())),
		MessageBody: aws.String(*message.Body),
		QueueUrl:    &queueUrl,

	})
	if err != nil {
		common.Warning("[BROKER] Errore nell'invio del messaggio. " + err.Error())
		return err
	}

	return nil

}

//Funzione che elimina una coda
func deleteQueue(queueUrl string) (retErr error) {

	//Creo un service client SQS
	svc := sqs.New(common.Sess)

	common.Info("[BROKER] Eliminazione della coda: " + queueUrl)

	//Elimino la coda
	_, err := svc.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueUrl),
	})

	if err != nil {
		common.Warning("[BROKER] Errore nell'eliminazione della coda. " + err.Error())
		return err
	}

	common.Info("[BROKER] Coda eliminata con successo")

	return nil
}



//Funzione che elimina un subscriber dal sistema
func deleteSubscriber(id string) (retErr error) {

	var err error = nil

	//Ottengo l'URL della coda
	queueUrl, err := getQueueUrl(id)
	if err != nil {
		common.Warning("[BROKER] Errore nell'ottenere l'URL della coda. " + err.Error())
		//return errors.New("error obtaining queue")
	}

	common.Info("[BROKER] La coda da eliminare per il subscriber: " + id + " è: " + queueUrl)

	//Eliminazione della coda
	err = deleteQueue(queueUrl)
	if err != nil {
		common.Warning("[BROKER] Errore nell'eliminazione della coda. " + err.Error())
	} else {
		common.Info("[BROKER] Coda rimossa con successo")
	}

	//Eliminazione della entry dal DB
	err = removeEntryDB(id)
	if err != nil {
		common.Warning("[BROKER] Errore nell'eliminazione della entry sul DB. " + err.Error())
		return errors.New("error in removing item in dynamodb")
	} else {
		common.Info("[BROKER] Rimozione subscriber " + id + " effettuata con successo.")
	}

	return nil

}
