# DGDS
Distributed Gathering Detection System
Andrea Paci - 0286387

# H1 Progetto SDCC 2019/2020 - Andrea Paci 

I requisiti per eseguire l'applicativo e istanziarlo su AWS sono:
 - Macchina Unix dotata di terminale
 - Docker
 - Un account AWS
 - aws cli
 - eb cli (Elastic beanstalk cli)

Inoltre per eseguire la dashboard di controllo è necessario avere un browser aggiornato, consigliabile l'uso di Google Chrome o Firefox.
 
Se non si riesce a visualizzare/utilizzare correttamente la dashboard con il browser, è possibile comunque emulare lo stesso comportamento usando un client per effettuare chiamate API REST

Innanzitutto è necessario copiare le **credenziali di accesso ad AWS nelle variabili d'ambiente nei Dockerfile** presenti nelle cartelle "broker", "publisher" e "subscriber" presenti in "Sorgente". 
*NOTA BENE: l'account (o il ruolo IAM) selezionato deve essere ROOT o deve possedere i pieni permessi per:
 - Elastic Beanstalk
 - Amazon SQS
 - DynamoDB
 - EC2


Per iniziare l'installazione, aprire un terminale, porsi in questa cartella ed eseguire il comando:

> $ sh ./start.sh

Lo script creerà l'infrastruttura su AWS e nella regione specificata.
Se si vuole cambiare la regione, è sufficiente cambiarla nei file "config.json" e cambiarla nello "script.sh", posta come variabile d'ambiente

Per connettersi al logger remoto, il quale riporterà gli eventi del sistema e permette di monitorare l'invio e la ricezione di messaggi è necessario eseguire netcat su una console con il comando

> $ nc hostremotelogger:60001

e successivamente è necessario inviare il carattere "l" per notificare il fatto di entrare in modalità listening

*NOTA: "hostremotelogger" viene fornito solo dopo che si istanzia il logger su Elastic Beanstalk
*NOTA#2 A causa della politica del load balancer imposta sulle connessioni persistenti, la connessione con il remote logger terminerà in un tempo breve. E' sufficiente riconnettersi ed inviare il carattere "l"


Esistono due file di configurazione:
 - conf_db.json
 - config.json


Il primo esporta i parametri di configurazione su un Database DynamoDB. Questi valori sono necessari per il Broker. Questi valori possono essere cambiati successivamente tramite la dashboard fornita (Il sito web contenuto nella casella Dashboard) oppure direttamente attraverso l'interfaccia di AWS DynamoDB.
(La spiegazione degli stessi è presente nella cartella "Sorgente/broker/broker-configuration.go"

Il secondo invece sono configurazioni che vengono salvate in locale:
	- **LoggerHost**: rappresenta la combinazione hostname:porta per connettersi al logger remoto per inviare le informazioni
	- **AwsBroker**: Hostname del broker
	- **RetryDelay**: Intervallo di tempo tra un tentativo di connessione e un altro in caso di fallimenti
	- **PoistDelay**: Intervallo di attesa usato dal Subscriber per inviare l'aggiornamento della posizione 
	- **OpDelay**: Intervallo di tempo usato per ritardare alcune operazioni 
	- **SimulationTime**: tempo di simulazione del publisher/subscriber nel caso di una esecuzione non interattiva
	- **RcvMessDelay**: tempo tra una interrogazione e un altra a SQS
	- **PollingTime**: tempo di polling per la ricezione (vedere Long Polling SQS per maggiori informazioni
	- **MaxRcvMessage**: numero di messaggi massimo che si possono ricevere con una singola interrogazione a SQS
	- **Region**: regione di AWS


E' possibile eseguire il publisher/subscriber in modalità sia interattiva che non. Per fare ciò è necessario porsi nelle cartelle contenutenenti il codice sorgente del publisher/subscriber ed eseguire: 

> $ docker build -t nome_applicazione .
> $ docker run nome_applicazione

Per eseguire gli applicativi in maniera interattiva, cambiare il secondo parametro presente nei rispedivi Dockerfile con il carattere "i".

Gli applicativi supportano anche altri parametri in ingresso da linea di comando (gli args sono presentati in ordine):

 - Subscriber:
	- interactive = (se impostato su 'i' allora interattivo, altrimenti no)
	- subscriber ID = id del subscriber (il subId deve essere già registratogià registrato	
	- Coda ricezione := Url coda SQS per ricevere messaggi (è possibile specificare una coda qualsiasi)
	- Coordinata X: coordinata X del sub
	- Coordinata Y: coordinata Y del sub
	- Topics: lista di topic a cui ci si vuole iscrivere (inserire un topic per argomento)


 - Publisher: 
	- interactive = (se impostato su 'i' allora interattivo, altrimenti no)
	- Name = Nome struttura
	- Topic = topic a cui mandare il messaggio
	- peopleNum = persone nella struttura
	- Coordinata X: coordinata X del sub
	- Coordinata Y: coordinata Y del sub
	- Raggio: Raggio di pubblicazione del messaggio
	- Metri quadri: metri uqdri della struttura

Modificando i dockerfiles è possibile usare i parametri in ingresso


E' possibile eseguire sia il broker che il remote logger anche in locale, è sufficiente cambiare il nome di entrmbi gli host nel file di configurazione



	
