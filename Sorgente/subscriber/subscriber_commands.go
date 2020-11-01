package main

import (
	"common"
	"strconv"
)

/*
			subscriber_commands.go

	Questo modulo esporta le funzionalità per la comuncazione diretta con il sistema
		per effettuare registrazione, topic subscribe, topic unsubscribe e
		deregistrazione dal sistema.
	La comunicazione avviene attraverso API REST

*/



func register() (id string, queue string, retErr error){

	common.Info("[SUB] Registrazione del subscriber")

	statusCode, resp, err := common.PutRequest(common.Config.AwsBroker + "/subscriber", nil, common.SubRegistrationResponse{})
	if err != nil {
		common.Fatal("[SUB] Errore nella registrazione ( " + strconv.Itoa(statusCode) + " ). " + err.Error())
		return "", "", err
	}
	response := common.SubRegistrationResponse{}
	common.FillStruct(&response, resp.(map[string]interface{}))


	subID := response.SubID
	recvQueue := response.QueueURL

	common.Info("[SUB] Subscriber registrato correttamente ( " + strconv.Itoa(statusCode) + " ): " + subID + "; Coda: " + recvQueue)

	return subID, recvQueue, nil



}


func updateSubscriberPosition(subId string, positionX int, positionY int) (retErr error){

	common.Info("[SUB] Aggiornamento posizione subscriber")

	statusCode, _, err := common.PostRequest(common.Config.AwsBroker + "/subscriber/" + subId + "/position", common.SubPositionUpdateRequest{PositionX: strconv.Itoa(positionX), PositionY: strconv.Itoa(positionY)}, nil)
	if err != nil {
		common.Fatal("[SUB] Errore nell'aggiornamento della posizione' ( " + strconv.Itoa(statusCode) + " ). " + err.Error())
		return err
	}

	common.Info("[SUB] Posizione aggiornata con successo ( " + strconv.Itoa(statusCode) + " ): " + subId + "; Posizione: [ " + strconv.Itoa(positionX) + ", " + strconv.Itoa(positionY) + "]")

	sendLogMessage(subId, "Posizione aggiornata con successo: [ " + strconv.Itoa(positionX) + ", " + strconv.Itoa(positionY) + "]")

	return nil

}




func subscribeTopic(subId string, topics []string) (retErr error){

	common.Info("[SUB] Topic subscribe")

	statusCode, _, err := common.PutRequest(common.Config.AwsBroker + "/subscriber/" + subId + "/topic", common.SubTopicSubscribeRequest{Topics: topics}, nil)
	if err != nil {
		common.Fatal("[SUB] Errore nella aggiunta di topic ( " + strconv.Itoa(statusCode) + " ). " + err.Error())
		return err
	}


	common.Info("[SUB] Topic aggiunti con successo ( " + strconv.Itoa(statusCode) + " ): " + subId + "; Topics: [ " + common.ConcatenateArrayValues(topics, ", ") + " ]")

	sendLogMessage(subId, "Iscrizione ai topic con successo: [ " + common.ConcatenateArrayValues(topics, ", ") + " ]")


	return nil

}




func unsubscribeTopic(subId string, topics []string) (retErr error){

	common.Info("[SUB] Topic unsubscribe")

	statusCode, _, err := common.DeleteRequest(common.Config.AwsBroker + "/subscriber/" + subId + "/topic", common.SubTopicSubscribeRequest{Topics: topics}, nil)
	if err != nil {
		common.Fatal("[SUB] Errore nella rimozione di topic ( " + strconv.Itoa(statusCode) + " ). " + err.Error())
		return err
	}


	common.Info("[SUB] Topic rimossi con successo ( " + strconv.Itoa(statusCode) + " ): " + subId + "; Topics: [ " + common.ConcatenateArrayValues(topics, ", ") + " ]")

	sendLogMessage(subId, "Rimozione di topic con successo: [ " + common.ConcatenateArrayValues(topics, ", ") + " ]")

	return nil
}



func unsubscribe(subId string) (retErr error){

	common.Info("[SUB] Rimozione subscriber")

	statusCode, _, err := common.DeleteRequest(common.Config.AwsBroker + "/subscriber/" + subId, nil, nil)
	if err != nil {
		common.Fatal("[SUB] Errore nella rimozione del subscriber ( " + strconv.Itoa(statusCode) + " ). " + err.Error())
		return err
	}

	common.Info("[SUB] Subscriber rimosso con successo ( " + strconv.Itoa(statusCode) + " ): " + subId + ".")

	sendLogMessage(subId, "Deregistrazione con successo.")

	return nil
}


