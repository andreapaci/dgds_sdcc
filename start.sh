#!/bin/bash

export AWS_REGION=us-east-1

echo "Prima di eseguire lo script assicurarsi di aver letto il README e di avere tutti i requisiti. Se si intende modificare i file di configurazione farlo prima di eseguire lo script.
Per continuare premere ENTER (CTRL-C per bloccare)"

read in


echo "Creazione dei database su DynamoDB
"


echo "Creazione tabella con i parametri di configurazione ...
"
aws dynamodb create-table --table-name configuration --attribute-definitions AttributeName=FieldName,AttributeType=S --key-schema AttributeName=FieldName,KeyType=HASH --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5


echo "
Popolazione tabella di configurazione ...
"
sleep 4

aws dynamodb batch-write-item --request-items file://conf_db.json


echo "
Creazione della tabella subscribers ...
"

aws dynamodb create-table --table-name subscriber --attribute-definitions AttributeName=SubID,AttributeType=S --key-schema AttributeName=SubID,KeyType=HASH --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 

echo "Esportazione logger remoto su Elastic Beanstalk

--------------------------------------------
[Creazione Remote logger sulla porta 60001]

Rispondere alle cose chieste come segue: 
 Regione: (Quella impostata nel file config.json, di default è us-east-1)
 Applicazione: remote_logger (o lasciare il nome di default)
 Tipo applicazione: Go
 Platform branch: Amazon Linux 2 64bit
 SSH: N
 Environment name e CNAME: lasciare default (premere ENTER)
 Spot Fleet: N
 Crea nuova applicazione 

"

cd Sorgente/remotelogger
eb init
eb create --elb-type classic -ix 1

echo "Remote logger stanziato"

eb status | grep "CNAME: "

echo "Copiare il nome dell'host nel file config.json sostituendo xxxxxx 
Premere ENTER dopo aver copiato il nome dell'host"

read in1


cd ../..

cp config.json Sorgente/broker/

echo "Esportazione Broker su Elastic Beanstalk


--------------------------------------------
[Creazione Broker]

Rispondere alle cose chieste come segue: 
 Regione: (Quella impostata nel file config.json, di default è us-east-1)
 Applicazione: broker (o lasciare il nome di default)
 Tipo applicazione: Dokcer
 Platform branch: Amazon Linux 2 64bit
 SSH: N
 Environment name e CNAME: lasciare default (premere ENTER)
 Spot Fleet: N
 Crea nuova applicazione  

"

cd Sorgente/broker
eb init
eb create --elb-type classic


echo "Broker stanziato"

eb status | grep "CNAME: "

echo "Copiare il nome dell'host nel file config.json sostituendo yyyyy 
Premere ENTER dopo aver copiato il nome dell'host"

read in2


cd ../..


echo "Copio i file di configurazione all'interno degli applicativi del subscriber e del publisher"

cp config.json Sorgente/subscriber/
cp config.json Sorgente/publisher/

echo "Sistema istanziato correttamente.
E' possibile provare il funzionamento del sistema eseguendo nelle cartelle di publisher e subscriber i comandi: 
docker build -t nome_applicazione .
docker run nome_applicazione
Nota: Per l'esecuzione dei publisher e dei subscriber rifarsi al README

Per verificare che il broker sia in funzione provare a connettersi tramite browser al sistema usando il nome dell'host (il CNAME) e verificare che la richiesta va a buon fine
Alternativamente si può usare il sito web presente nella cartella Dashboard e provare a fare una query qualsiasi, se il broker è offline non sarà possibile interagire con esso


Per verificare il funzionamento del logger remoto è invece sufficiente connettersi ad esso con netcat (comando presentato nel read me)"




