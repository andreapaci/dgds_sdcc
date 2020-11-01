package main

import (
	"common"
	"math/rand"
	"strconv"
	"time"
)

/*
			position_module.go

	Questo è il modulo che si occupa di gestire il retrive della posizione del subscriber

*/

//Simulazione di componente hardware/software che ottiene la posizione di un dispositvo
func position_update(subId string, strPositionX string, strPositionY string) (){

	var positionX int
	var positionY int

	positionX, err1 := strconv.Atoi(strPositionX)
	positionY, err2 := strconv.Atoi(strPositionY)

	if err1 == nil || err2 == nil {
		err := updateSubscriberPosition(subId, positionX, positionY)
		if err != nil {
			common.Warning("[SUB] Errore nell'aggiornamento della posizione. " + err.Error())
		}


	}


	//Posizione mockata//
	for {

		time.Sleep(time.Second * time.Duration(common.Config.PositDelay))

		newPosX, newPosY := getPosition()
		positionX = newPosX
		positionY = newPosY

		err := updateSubscriberPosition(subId, positionX, positionY)
		if err != nil {
			common.Warning("[SUB] Errore nell'aggiornamento della posizione. " + err.Error())
		}



	}

}

//Funzione che simula la ricezione della posizione
func getPosition() (positionX int, positionY int){


	newPosX := rand.Intn(common.TestPositionSize)
	newPosY := rand.Intn(common.TestPositionSize)

	return newPosX, newPosY

}