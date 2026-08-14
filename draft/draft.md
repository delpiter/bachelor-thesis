# Servizio di diagnotica della latenza in una rete anycast
## Introduzione
Il presente documento descrive le attività svolte da Foschi Gioele durante il periodo di tirocinio aziendale presso FlashStart, sotto la supervisione del Prof. Mirko Viroli, i Tutor Aziendali Francesco Collini e Stefano Babini. 
Il progetto è stato incentrato sulla modernizzazione architetturale e sul re-engineering ad alte prestazioni dei servizi di misurazione della latenza in una ***rete anycast*** (add definition).
Il sistema preesistente basato su architettura PHP/Apache e script di ***fan-out*** (add definition) Python soffriva di limitazioni prestazionali, elevato overhead di CPU/memoria ed eccessiva rigidità nella gestione dell'I/O concorrente.
## Background
// Maybe move company context up before the project definition.

> Contesto aziendale

FlashStart è una azienda che da oltre 20 anni protegge gli utenti nel mondo digitale con soluzioni di sicurezza informatica attraverso il filtraggio dei contenuti basato su DNS e intelligenza artificiale.
L'azienda fornisce filtraggio dei contenuti per utenti singoli o organizzazioni di ogni dimensione.
FlashStart opera su una rete Anycast globale composta da più di 160 nodi distribuiti strategicamente in tutti i continenti, progettata per garantire elevata disponibilità, latenza minima e continuità operativa anche in caso di interruzioni o picchi di traffico.

A partire dal 2026 FlashStart ha iniziato il più grande redesign dell'azienda, spostandosi da una applicazione con architettura monolitica ad un applicativo moderno con una architettura basata su microservizi altamente scalabili.
In questo documento si farà riferimento ai due applicativi come: ***FlashStart Cloud***, il sistema attualmente in uso ma destinato alla dismissione, e ***FlashStart Internet Protection 2026***, la soluzione individuata per la sua sostituzione.

Nel seguito della trattazione, per semplicità, si farà riferimento al nuovo sistema come *FlashStart 2026 (FS26)*.

> Importanza del tool.

Il tool di controllo latenza è uno strumento fondamentale per il Network Admin in quanto consente di calcolare a priori la latenza di risposta del DNS (in millisecondi) che l’utilizzatore avrà utilizzando il servizio di protezione della navigazione.

Essendo la protezione FlashStart “at DNS level”, una bassa latenza (tipicamente inferiore ai 10ms) è essenziale per garantire sicurezza e fluidità della navigazione in Internet allo stesso tempo.

Nel tool vengono anche stimati gli eventuali nodi alternativi in caso di problematiche (outage) del nodo preferenziale (best latency path). Essendo la rete basata su BGP Anycast, gli indirizzi IP del servizio di DNS sicuro sono presenti (annunciati) in ogni datacenter ed in caso di criticità del nodo principale, automaticamente l’utilizzatore continuerà a navigare filtrato e protetto su un nodo alternativo, con una latenza leggermente superiore.

### Architettura legacy del servizio

Il servizio latency si compone di tre servizi separati:
- Un upstream orchestrator
- Diversi downstream, uno per ciascuno dei servizi di DNS resolver all'interno dei 160+ server della rete anycast.
- La route `php` utilizzata da *FlashStart Cloud*.

#### Upstream Orchestrator
L'upstream orchestrator non è altro che un servizio dispiegato all'interno di ciascun DNS resolver.
È composto da due script, un controller php che gestisce le chiamate da parte dei client e uno script python esterno che gestisce il fan-out (Fan-out generally refers to the act of spreading out from a single point or source to multiple destinations) verso gli endpoint downstream e l'aggregazione dei risultati.

L'upstream orchestrator è inoltre responsabile del recupero della lista dei server da interrogare.

#### Downstream 
Il servizio di downstream è composto da un controller php che esegue comandi shell di sistema `ping/ping6` gestendo l'output come stringa rigida e da un parser.

```sh
ping -c 3 ipv4Address 2>&1
# or 
ping6 -c 3 ipv6Address 2>&1
```

Il parsing dell'output si basa su indici di riga fissi (8 righe)
```
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=111 time=71.1 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=111 time=53.9 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=111 time=42.8 ms

--- 8.8.8.8 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2000ms
rtt min/avg/max/mdev = 42.838/55.945/71.057/11.607 ms
```
Il parser si aspetta come input una stringa con questo formato specifico andando a leggere precisamente la settima riga ricavando il tempo medio di risposta attraverso semplici operazioni con le stringhe.
Questo metodo è altamente vulnerabile alle variazioni del sistema operativo target.

#### Route PHP
La route `php` riceve i dati dall'upstream orchestrator e inserisce informazioni aggiuntive riguardanti i server; Ha inoltre il compito di filtrare le risposte duplicate, causate da un dispiegamento di più DNS resolver in un unico firewall.

L'oggetto finale che viene restituito al client è il seguente:

```
[
    {
        "name": "IT-Cesena",
        "country_code": "IT",
        "ip": "192.168.0.1",
        "X": 30.2681,
        "Y": , 60.8518,
        "color": "#5AB500",
        "ping": 40,
        "load": 0
    }
]
```

[Deployment diagram del servizio].
[Grafici del flusso dell'informazione].
## Analisi
In questo capitolo vengono evidenziati i requisiti del progetto, analizzando i limiti imposti dalla quantità di risorse a disposizione da ciascun server all'interno della rete.
### Obbiettivi
L'obbiettivo di questo progetto è quello di riscrivere un servizio di monitoraggio della latenza, partendo da un codice monolitico scritto in php e python.
La riscrittura coinvolge 3 sezioni distinte:
- Servizio downstream
    - Servizio localizzato all'interno di ciascuno dei server remoti nella rete utilizzato per richiedere il tempo di latenza tra il server e una macchina specifica in una qualsiasi location nel mondo.  
- Servizio upstream
    - Servizio localizzato all'interno del server principale, questo servizio ha il compito di recuperare la lista dei server attivi da interrogare per la latenza, aggregare e riordinare i risultati ottenuti.

Per questi due progetti gli obbiettivi sono i seguenti:
- Eliminazione dei colli di bottiglia: Sostituzione dei processi di sistema shell-based con esecuzione controllata e concurrent fan-out.
- Scalabilità orizzontale: Standardizzazione dei container Docker light-weight e orchestrazione con Docker-Compose per ambienti di test e produzione.
- Garanzia di compatibilità (Golden Testing): Mantenimento rigoroso dei contratti API REST sia per i client dashboard sia per i servizi downstream/upstream legati all'infrastruttura MySQL e PostgreSQL.

- Integrazione del servizio con *FlashStart 2026*. 
    - In questa sezione dovranno essere riprogettati sia l'interfaccia grafica che la route `http` per l'esecuzione del servizio. 
### Limitazioni
Per quanto riguarda i servizi di downstream e upstream le limitazioni sono le seguenti:
- Il software deve essere il più leggero e efficiente possibile, in quanto dovrà essere dispiegato su server che devono gestire milioni di richieste giornaliere. Il software non deve assolutamente rallentare il funzionamento delle macchine fisiche.
- Il software deve essere retrocompatibile con la versione precedente, in modo tale che possa essere utilizzato dalla dashboard legacy senza necessità di ulteriori modifiche al codice attuale.
### Analisi dei Requisiti
I due servizi principali sono stati progettati seguendo il pattern di programmazione MVC.
- Devono esporre delle api ben definite (View), analizzare le richieste ricevute (Controller) e eseguire le operazioni sui dati (Model).
#### Upstream service
Il servizio di upstream deve interrogare tutti i server e validare i valori di output.
Deve esporre due interfaccie API:
- `GET api/latency/{network}`
- `GET api/latency/cloud/{network}`

La risposta ottenuta dalle due interfacce deve essere omogenea.
```json
[
    ["<ipAddress>", "latencyValue"],
    ["<ipAddress>", "latencyValue"],
    ...
]
```
La struttura del messaggio di risposta è stata riportata dal tipo di ritorno della vecchia API.

La richiesta si divide in due sezioni distinte

> Recupero della lista dei client

In base alla richiesta ricevuta viene selezionata una lista diversa di server da interrogare.
- `api/latency/` seleziona gli host fisici attualmente attivi.
- `api/latency/cloud` seleziona una lista di host dispiegati sul cloud.

[TODO] check if correct

Il fallimento della richiesta o il superamento di una deadline pre-impostata comporta il fallimento della richiesta di latenza.

> Aggregazione dei valori

In questa sezione viene validato ciascun indirizzo ip ottenuto in precedenza; Per ogni host valido viene interrogato il server server downstream corrispondente per il valore della latenza.
Ogni richiesta deve essere trattata in maniera indipendente dalle altre: il fallimento di una richiesta non deve compromettere l'intera aggregazione.
Successivamente vengono normalizzati i risultati ottenuti per aderire al modello di risposta analizzato in precedenza.
- Se una richiesta fallisce, impiega troppo tempo o ritorna un codice http diversa da `2xx`, comporta l'assegnazione del valore `-1` alla richiesta a quel server.

Infine la lista di valori deve essere ordinata in ordine ascendente secondo la latenza.
#### Downstream service
Il servizio di downstream comprende 3 sezioni:
- Esposizione dell'interfaccia API per la richiesta della latenza.
`GET /api/latency`
La richiesta dovrà ritornare un semplice valore che rappresenta il tempo di risposta medio del server interrogato.

Una richiesta valida deve contenere un parametro `network` nella richiesta `http` e come valore deve essere presente un indirizzo ip valido sia ipv4 che ipv6.
- Appena ricevuta la richiesta, essa deve essere validata, una richiesta con indirizzo assente o non valido deve essere rifiutata immediatamente e ritornare il codie `403`.

Una volta validato l'indirizzo viene eseguito il ping della macchina specificata, questo passaggio deve essere indipendente dal tipo di versione dell'indirizzo che è stato ricevuto.
- Questa operazione ha delle deadline strette per evitare di consumare troppe risorse, se una richiesta ping eccede questa deadline il pacchetto viene considerato perso.
- Se in un qualsiasi momento accade un errore, la richiesta non deve fallire, invece si deve attivare un meccanismo di fallback e ritornare un valore di default (`-1`).

Infine deve essere fatta la validazione del risultato ottenuto dall'esecuzione del ping.
- Se la perdita di pacchetti blocca il calcolo del tempo di risposta medio, la richiesta deve tornare `-1`.

#### Integrazione con FlashStart 2026
Per permettere al frontend di mostrare i dati aggregati, è prima necessario uno passo intermedio.
È necessario aggiungere una nuova API ai microservizi attuali, questa API avrà il compito di autenticare l'utente che richiede il servizio di latenza, interpellare il servizio di upstream e ricevere la risposta e successivamente recuperare le informazioni necessarie per il display delle informazioni all'interno della mappa come:
- Le geolocalizzazione (latitudine, longitudine, nome città e nazione) di ciascun server interrogato.
- La geolocalizzazione dell'indirizzo (latitudine e logitudine a livello di nazione) ip dato dal client al momento della richiesta.

L'integrazione con l'applicativo dovrà essere coerente con la codebase già presente.
## Progettazione
In questo capitolo verranno speigate in maniera dettagliata le decisioni architetturali prese per lo sviluppo dei due serivzi principali.
### Downstream Service
Il servizio di downstream deve essere in grado di gestire in maniera trasparente l'utilizzo di versioni del protocollo IP differenti (`ipv4` e `ipv6`).
Le operazioni specifiche ad una versione del protocollo `ip` verranno delegate ad una classe che, attraverso l'utilizzo del pattern *strategy*, sarà in grado di gestire le varie operazioni necessarie all'invio del messaggio `ICMP` in base alla verisone del protocollo ip utilizzata.

``` bibtex
@book{gamma1994design,
  author    = {Gamma, Erich and Helm, Richard and Johnson, Ralph and Vlissides, John},
  title     = {Design Patterns: Elements of Reusable Object-Oriented Software},
  year      = {1994},
  publisher = {Addison-Wesley},
  address   = {Reading, MA},
  isbn      = {0-201-63361-2}
}
```

### Upstream Service
Il numero di server da interrogare potrebbe aumentare notevolmente nel tempo; Eseguire le chiamate in sequenza non è possibile, in quanto con un numero elevato di interrogazioni da fare l'esecuzione verrebbe sicuramente interrotta dal timeout imposto.
È necessario quindi eseguire le chiamate in parallelo.
L'operazione è di tipo *embarassingly parallel* in quanto ogni singola esecuzione è indipendente dalle altre, sarebbe dunque facilmente parallelizzabile assegnando un thread a ciascuna richiesta da fare.
Questo però andrebbe a consumare una elevata quantità di risorse, e ciò violerebbe il vincolo posto in precedenza (capitolo analisi).
È stato scelto di limitare il numero di thread lavoratori sfruttando il _bounded parallelism pattern_.
### Pattern Comuni
Entrambi i servizi comprendono la stessa logica per 2 concetti importanti:
- Il caricamento dei valori di configurazione
- La gestione di middleware nelle richieste http.
#### Gestione dei middleware
Si vuole poter eseguire multipli middleware in sequenza a seguito di una richista http per gestire il logging strutturato, il disaster recovery ed altri eventuali controlli.
Il sistema per la gestione dei middleware in sequenza utilizza il design pattern *chain of responsibility* che permette di passare la richiesta lungo una catena di "handler".
Al ricevimento di una richiesta dopo averla processata, un handler può decidere se passarla al prossimo handler all'interno della catena o se interrompere l'esecuzione.
#### Caricamento delle variabili di configurazione ??
Le variabili di configurazione vengono caricate attraverso variabili d'ambiente all'interno della macchina.
Ciascun servizio ha una serie di configurazioni obbligatorie e una serie di configurazioni opzionali.
Viene definita una classe con una serie di metodi per fare il parsing del valore delle variabili d'ambiente;
Il fallimento di una di queste funzioni, dato da un valore non coerente con il tipo richiesto o la mancata presenza di una variabile obbligatoria risulta nell'immediata terminazione del programma.
### Integrazione con FlashStart 2026
Per integrare il servizio di latenza con FlashStart2026 è necessaria l'aggiunta di 
Si vuole permettere agli utilizzatori di *FlashStart 2026* di usufruire del servizio di latenza.
È dunque necessario aggiungere un entry point all'interno dell'applicazione.
Per mantenere una codebase coerente con il resto dell'applicativo sviluppato dagli altri developer in azienda, è stato scelto di progettare il nuovo entrypoint con il pattern `M.V.C.`.
## Sviluppo
Durante questa fase sono stati analizzati i linguaggi da utilizzare per la ricostruzione del servizio.
Per rispettare i requisiti del problema (servizio efficiente e a basso consumo di risorse), il linguaggio selto deve essere un linguaggio compilato e non interpretato, linguaggi come python e php sono stati scartati a priori.
Il secondo requisito è la possibilità di gestire codice in maniera concorrente.
Dati questi due principali requisiti è stato scelto `go` come linguaggio di scrittura del servizio lato backend, poiché è un linguaggio compilato e soprattutto supporta in maniera nativa e facilitata la gestione di routine concorrenti, concetto fondamentale per lo sviluppo degli strumenti richiesti.
### Implementation Highlight
#### Esecuzione del Ping
L'esecuzione del ping sul server è l'operazione più importante del sistema.
La versione legacy del software eseguiva semplicemente il comando `ping -c MAX_PING <network>` su una shell bash creata sul momento ed eseguiva poi il parsing del risultato. Il parser si aspetta un formato molto specifico del risultato, il ché rendeva il sistema molto fragile ad errori, un minimo cambiamento al formato standard della risposta del comando ping eseguito su una shell linux e la risposta veniva considerata come un fallimento.

Il nuovo sistema è stato creato con l'affidabilità alla base.
Al posto di eseguire una shell bash, è stata utilizzata la libreria di `golang.org/x/net/icmp` per eseguire chiamate `ICMP` attraverso la rete; A questa libreria è stato creato un wrapper che permetta l'utilizzo facilitato di una libreria altrimenti altamente complessa.

Il wrapper permette di eseguire una serie di messaggi `ICMP:Echo` con id univoci in modo da filtrare pacchetti `ICMP` non voluti, attraverso il pattern strategy definito in precedenza, permette di inviare in maniera trasparente la richiesta sia su una rete IPv4 che su una rete IPv6, in base all'indirizzo fornito dall'utente. Infine permette opzionalmente la possibilità di impostare il `TTL` ad ogni pacchetto.
``` go
// insert example code
```
Quest'ultima feature permette una potenziale futura espansione del servizio attuale con l'aggiunta del traceroute.

Questo wrapper risolve il problema della fragilità del risultato, poiché il parsing del risultato non viene più fatto in base all'output di un comando di una bash. Il risultato finale viene calcolato sulla base dei risultati di ogni pacchetto che vengono salvati in locale.

#### Bounded Parallelism
Il linguaggio `golang` mette a disposizione due costrutti nativi per la gestione di routine parallele: le goroutine, funzioni parallele asincrone e i canali (`chan`) strutture dati in grado di gestire automaticamente la concorrenza senza provocare race conditions.
I canali possono anche essere visti come dei semafori nella teoria della programmazione concorrente.

L'implementazione del bounded parallelism pattern si è basata da una funzione citata in un blog nel sito ufficiale di `golang` che calcola il *checksum MD5* di tutti i file in una directory specificata, riadattandola secondo le necessità del sistema in sviluppo.

``` bibtex
@misc{ajmani2014pipelines,
  author       = {Ajmani, Sameer},
  title        = {Go Concurrency Patterns: Pipelines and Cancellation},
  howpublished = {The Go Blog},
  year         = {2014},
  month        = {March},
  url          = {https://go.dev/blog/pipelines},
  note         = {Licensed under CC BY 4.0. Accessed: \today}
}
```

Il funzionamento dell'algoritmo si basa su 3 canali:
- Un canale che eroga uno alla volta ciascun indirizzo dei server da interrogare.
- Un canale che colleziona i risultati.
- Un canale che segnala la conclusione.

Una volta inizializzati i canali viene creato l'insieme di "thread worker" che eseguiranno le richieste di latenza ai vari server.
A ciascun worker vengono condivisi i tre canali creati, quando riceve un nuovo indirizzo IP dal canale, esegue la chiamata, ritorna il valore di latenza e si rimette in ascolto per il prossimo indirizzo.

``` go
func serverListWalk(
    serverIp []string
) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)
	go func() {
		defer close(paths)

		for i := range serverIp {
			select {
			case paths <- serverIp[i]:
			case <-done:
				errc <- errors.New("Request cancelled")
				return
			}
		}
		errc <- nil
	}()
	return paths, errc
}
```

``` go
func worker(
	done <-chan struct{},
	servers <-chan string,
	c chan<- result,
	targetIp string,
	fetchLatency func(serverIp, targetIp string) (Result, error),
) {
	for serverIp := range servers {
		latencyResult, err := fetchLatency(serverIp, targetIp)
		select {
		case c <- result{latencyResult.Address, latencyResult.Value, err}:
		case <-done:
			return
		}
	}
}
```
#### Frontend
Per lo sviluppo del frontend è stato coinvolto un esperto di user experience in modo tale da mantenere un'alto standard di usabilità e coerenza di stile con il resto dell'applicativo.
La progettazione è partita dal design della vechia pagina, la nuova doveva riproporre le stesse feature in maniera più chiara e pulita.

[wireframe del modello]

Inizialmente è stato proposto un design semplice che riproponeva la pagina precedente, semplicemente modificando lo stile e rendendolo più moderno.

[mockup iniziale]

Dopo l'intervento dell'esperto il mockup si è evoluto dando la possibilità alla pagina di trasmettere più informazioni, nonostante il mantenimento delle stesse API.

[mockup finale]

Questo design finale mostra i dati in maniera chiara con uno stile riadattato al resto dell'applicazione, consentendo inoltre opzioni per nuove aggiunte future.
## Deployment
### Infrastructure


Dopo una analisi della semantica delle informazioni fornite da questo servizio, siamo giunti alla conclusione che nessun microservizio attualmente presente sarebbe adatto al contenimento di questa nuova API, è stata quindi necessaria la creazione di un nuovo microservizio all'interno dell'applicativo: `infrastructure`.

Il nuovo microservizio servirà per tutte quelle nuove feature che implementano un qualsiasi tipo di tool di supporto all'utilizzo dell'applicazione.
Per la creazione del nuovo microservizio è stato necessario configurare diversi aspetti:
- Ambiente di testing delle api e di qualsiasi business logic interna al nuovo microservizio.
- Nuovo database direttamente collegato al microservizio
- Due dockerfile, uno per l'ambiente di development e testing e uno per l'ambiente di esercizio (production), necessario per poi dispiegare il nuovo microservizio all'interno di un cluster kuberneetes.
### Backend
Per il deploy dei servizi di downstream e upstream all'interno della infrastruttura esistente sono stati creati dei dockerfile ad hoc contenenti i file binari compilati dei due servizi.
Per evitare di caricare sui server codice binario inutilizzato (come il compilatore del linguaggio), è stato creato un dockerfile multistage
- Un primo stage che scarica le librerie necessarie e compila i codici sorgenti in un unico file binaro eseguibile.
- Un secondo stage che utilizzando una immagine leggera (`FROM debian:bookworm-slim`) elimina qualsiasi file non necessario e successivamente copia il file binario creato al primo stage, minimizzando le risorse necessarie per l'esecuzione del servizio.
## Analisi dei Risultati
### Testing automatico
### Sperimentazioni

## Conclusioni
### Lavori futuri

## Bibliografia