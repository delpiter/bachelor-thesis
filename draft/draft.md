# Servizio di diagnotica della latenza in una rete anycast
## Sommario
Il progetto è incentrato sulla modernizzazione architetturale e sul re-engineering dei 
servizi di misurazione della latenza in una rete anycast. Il sistema preesistente, 
basato su architettura PHP/Apache e script di fan-out Python, soffriva di limitazioni 
prestazionali, elevato overhead di CPU/memoria, una bassa affidabilità di risposta ed eccessiva rigidità nella gestione dell'I/O concorrente.

Il progetto ha coinvolto la riscrittura di due sotto-servizi principali. Il servizio di 
downstream, incaricato di calcolare la latenza tra il server stesso e la macchina 
specificata in input, era originariamente scritto in PHP ed è stato riprogettato in Go, 
adottando i principi della Clean Architecture e un design basato su design pattern, massimizzando l'affidabilità della risposta. Il servizio di upstream, responsabile dell'aggregazione dei dati raccolti dalle interrogazioni dei server di downstream, era originariamente scritto in Python ed è stato anch'esso riscritto in Go, sfruttando il modello di concorrenza nativo del linguaggio (goroutine e channel) per superare la rigidità nella gestione dell'I/O concorrente riscontrata nel sistema preesistente.

Il nuovo sistema è stato validato confrontandolo con l'architettura preesistente, 
mostrando un miglioramento dell'affidabilità del servizio e una riduzione dell'utilizzo 
di memoria sui server di dispiegamento. Il lavoro ha incluso inoltre la progettazione 
di un'interfaccia utente modernizzata per il nuovo applicativo.
## Introduzione
Il presente documento descrive le attività svolte durante il periodo di tirocinio aziendale presso FlashStart, sotto la supervisione del Prof. Mirko Viroli, i Tutor Aziendali Francesco Collini e Stefano Babini.

FlashStart è una azienda che da oltre 20 anni protegge gli utenti nel mondo digitale con soluzioni di sicurezza informatica attraverso il filtraggio dei contenuti basato su DNS e intelligenza artificiale.
L'azienda fornisce filtraggio dei contenuti per utenti singoli o organizzazioni di ogni dimensione.
FlashStart opera su una rete anycast globale (aggiungi definizione) composta da più di 160 nodi distribuiti strategicamente in tutti i continenti, progettata per garantire elevata disponibilità, latenza minima e continuità operativa anche in caso di interruzioni o picchi di traffico.

A partire dal 2026 FlashStart ha iniziato il più grande redesign dell'azienda, spostandosi da una applicazione con architettura monolitica ad un applicativo moderno con una architettura basata su microservizi altamente scalabili.
In questo documento si farà riferimento ai due applicativi come: ***FlashStart Cloud***, il sistema attualmente in uso ma destinato alla dismissione, e ***FlashStart Internet Protection 2026***, la soluzione individuata per la sua sostituzione.

Nel seguito della trattazione, per semplicità, si farà riferimento al nuovo sistema come *FlashStart 2026 (FS26)*.

Lo strumento di controllo latenza è uno strumento fondamentale per il Network Admin in quanto consente di calcolare a priori la latenza di risposta del DNS (in millisecondi) che l’utilizzatore avrà utilizzando il servizio di protezione della navigazione.

Essendo la protezione FlashStart "at DNS level", una bassa latenza (tipicamente inferiore ai 10ms) è essenziale per garantire sicurezza e fluidità della navigazione in Internet allo stesso tempo.

Nello strumento vengono anche stimati gli eventuali nodi alternativi in caso di problematiche (outage) del nodo preferenziale (best latency path). Essendo la rete basata su BGP Anycast, gli indirizzi IP del servizio di DNS sicuro sono presenti (annunciati) in ogni datacenter ed in caso di criticità del nodo principale, automaticamente l’utilizzatore continuerà a navigare filtrato e protetto su un nodo alternativo, con una latenza leggermente superiore.
```bibtex
// bgp
@techreport{rfc4271,
  author      = {Rekhter, Yakov and Li, Tony and Hares, Susan},
  title       = {A Border Gateway Protocol 4 (BGP-4)},
  institution = {IETF},
  year        = {2006},
  month       = {January},
  number      = {RFC 4271},
  url         = {https://www.rfc-editor.org/rfc/rfc4271},
  note        = {Internet Standards Track document}
}

// ip
@techreport{rfc791,
  author      = {Postel, Jon},
  title       = {Internet Protocol},
  institution = {DARPA},
  year        = {1981},
  month       = {September},
  number      = {RFC 791},
  url         = {https://www.rfc-editor.org/rfc/rfc791}
}

// icmp
@techreport{rfc792,
  author      = {Postel, Jon},
  title       = {Internet Control Message Protocol},
  institution = {DARPA},
  year        = {1981},
  month       = {September},
  number      = {RFC 792},
  url         = {https://www.rfc-editor.org/rfc/rfc792}
}

// http
@techreport{rfc2616,
  author      = {Fielding, Roy and Gettys, Jim and Mogul, Jeffrey and Frystyk, Henrik and Masinter, Larry and Leach, Paul and Berners-Lee, Tim},
  title       = {Hypertext Transfer Protocol -- HTTP/1.1},
  institution = {IETF},
  year        = {1999},
  month       = {June},
  number      = {RFC 2616},
  url         = {https://www.rfc-editor.org/rfc/rfc2616},
  note        = {Obsoleted by RFC 7230--7235}
}
```

> Struttura dei Capitoli

Capitolo 2 – Sfondo tecnologico e architettura: descrive nel dettaglio l'architettura del servizio e i suoi componenti chiave (upstream orchestrator, downstream e punto d'ingresso su FlashStart Cloud), evidenziandone le fragilità e le criticità da risolvere.

Capitolo 3 – Requisiti ed obiettivi: definisce i vincoli funzionali e strutturali stabiliti per ciascun elemento del sistema.

Capitolo 4 – Scelte di design: illustra le decisioni architetturali e di progettazione adottate per soddisfare i requisiti individuati nel capitolo precedente.

Capitolo 5 – Implementazione: motiva la selezione dei linguaggi di programmazione e presenta nel dettaglio lo sviluppo delle varie componenti.

Capitolo 6 – Deployment e integrazione: approfondisce le strategie e le decisioni operative per il dispiegamento delle nuove componenti nell'infrastruttura di rete e nel nuovo applicativo.

Capitolo 7 – Validazione e test: presenta le prove sperimentali condotte per verificare il corretto funzionamento dei componenti realizzati.

Capitolo 8 – Conclusioni: raccoglie le considerazioni finali sui risultati ottenuti e delinea i possibili sviluppi futuri.
## Background
### Architettura legacy del servizio
Il servizio latency si compone di tre servizi separati che comunicano in sequenza attraverso chiamate http per ottenere il risultato.

- **Upstream orchestrator**: componente centrale localizzato all'interno del server principale. Questa componente del servizio ha il compito di recuperare la lista dei server attivi da interrogare per la latenza, aggregare e riordinare i risultati ottenuti. 

- **Downstream**: Utilizzato per richiedere il tempo di latenza tra il server e una macchina specificata. Questa componente è dispiegata in ciascuno dei DNS resolver all'interno dei 160+ server della rete anycast.

- **La route `php`**: utilizzata da *FlashStart Cloud* per invocare il servizio ed esporre i dati raccolti.

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
La struttura di comunicazione deve rimanere quella attuale mantenendo le tre componenti descritte in precedenza.

Per le componenti di upstream e downstream gli obbiettivi sono i seguenti:
- **Eliminazione dei colli di bottiglia**: sostituzione dei processi di sistema shell-based con esecuzione controllata e concurrent fan-out*.

- **Scalabilità orizzontale**: Standardizzazione dei container Docker light-weight e orchestrazione con Docker-Compose per ambienti di test e produzione.

- **Garanzia di compatibilità**: Mantenimento rigoroso dei contratti API REST sia per i client dashboard sia per i servizi downstream/upstream legati all'infrastruttura MySQL e PostgreSQL.

Per l'integrazione del servizio con *FlashStart 2026* dovranno essere riprogettati sia l'interfaccia grafica che la route `http` per l'esecuzione del servizio all'interno del nuovo applicativo.

È inoltre richiesta la scrittura della documentazione dettagliata di ciascuna delle componenti sviluppate, in modo tale da facilitare sviluppi futuri.
### Limitazioni
Per quanto riguarda i servizi di downstream e upstream le limitazioni sono le seguenti:
- **Efficienza**: il software deve essere il più leggero e efficiente possibile, in quanto dovrà essere dispiegato su server che devono gestire milioni di richieste giornaliere. Il software non deve rallentare il funzionamento delle macchine fisiche.
- **Retrocompatibilità**: il software delle componenti di downstream e upstream deve essere retrocompatibile con la versione precedente, in modo tale che possa essere utilizzato dalla dashboard legacy senza necessità di ulteriori modifiche al codice attuale.

### Analisi dei Requisiti
I due servizi principali devono essere progettati seguendo il pattern di programmazione MVC.Devono esporre delle api ben definite (View), analizzare le richieste ricevute (Controller) e eseguire le operazioni sui dati (Model).
#### Upstream aggregator
Il servizio di upstream deve recuperare la lista di indirizzi ip dei server, interrogare tutti gli indirizzi e validare i valori di output.

Questo servizio deve esporre due interfacce API:
- `GET api/latency/{network}`: interfaccia utilizzata da altri servizi all'interno di *FlashStart Cloud*, non strettamente necessaria per la modernizzazione del servizio, ma mantenuta per la retrocompatibilità.
- `GET api/latency/cloud/{network}`: interfaccia utilizzata direttamente dal servizio di latency.

Il parametro del percorso `{network}` deve essere un indirizzo ip valido (sia ipv4 che ipv6) che corrisponde all'indirizzo della macchina su cui devono essere calcolati i tempi di risposta dei vari server.

La risposta ottenuta dalle due interfacce deve essere omogenea e deve inoltre corrispondere con la struttura originale del messaggio per mantenere la retrocompatibilità.
```json
[
    ["<ipAddress>", "latencyValue"],
    ["<ipAddress>", "latencyValue"],
    ...
]
```
Dove `<ipAddress>` è l'indirizzo del server downstream che ha risposto con il `latencyValue` corrispondente.

La richiesta si divide in due sezioni distinte

> Recupero della lista dei client

In base alla rotta chiamata deve essere selezionata una lista diversa di server da interrogare.
- `api/latency/` seleziona gli host fisici attualmente attivi.
- `api/latency/cloud` seleziona una lista di host dispiegati sul cloud.

Il fallimento della richiesta o il superamento di una deadline pre-impostata comporta il fallimento della richiesta di latenza.

> Aggregazione dei valori

In questa sezione deve essere validato ciascun indirizzo ip ottenuto in precedenza.
Per ogni host valido verrà interrogato il server server downstream corrispondente per ottenere il relativo valore della latenza rispetto all'indirizzo fornito dall'utente.
Ogni richiesta deve essere trattata in maniera indipendente dalle altre: il fallimento di una richiesta non deve compromettere l'intera aggregazione.
Una volta ottenuti, i risultati devono essere normalizzati per aderire al modello di risposta analizzato in precedenza.
Se una richiesta fallisce, impiega troppo tempo o ritorna un codice http diverso da `2xx`, deve essere assegnato il valore `-1` alla richiesta di quel server.

Infine la lista di valori deve essere ordinata in ordine ascendente secondo il valore della latenza.

#### Downstream
Il servizio di downstream deve esporre una unica interfacca API.
`GET /api/latency`.
Una richiesta valida deve contenere un parametro `network` nella richiesta `http` e come valore deve essere presente un indirizzo ip valido tra ipv4 e ipv6.
Appena ricevuta la richiesta, essa deve essere validata, una richiesta con indirizzo assente o non valido deve essere rifiutata immediatamente e ritornare il codie `403`.

La richiesta dovrà ritornare un semplice valore numerico che rappresenta il tempo di risposta medio di un ping dal server interrogato alla macchina con l'indirizzo ip dato dal campo `network` della richiesta.

È responsabilità dell'utente assicurasri che la macchina con l'indirizzo ip dato in input abbia il protocollo `ICMP` attivo.
- In caso non venga attivato, la richiesta deve fallire (`host unreachable`).

Una volta validato l'indirizzo viene eseguito il ping della macchina specificata, questo passaggio deve essere indipendente dal tipo di versione dell'indirizzo che è stato ricevuto.
Questa operazione ha delle deadline strette per evitare di consumare troppe risorse. La deadline deve essere derivata dal numero di pacchetti `ICMP` inviati più un piccolo buffer fisso.

Se in un qualsiasi momento accade un errore, la richiesta non deve fallire, invece si deve attivare un meccanismo di fallback e ritornare un valore di default (`-1`).

Infine deve essere fatta la validazione del risultato ottenuto dall'esecuzione del ping.
- Se la perdita di pacchetti (packet loss $> 50$ %) blocca il calcolo del tempo di risposta medio, la richiesta deve tornare `-1`.

#### Integrazione con FlashStart 2026
L'integrazione con *FlashStart 2026* serve per permettere al frontend dell'applicazione di mostrare i dati ottenuti.

Deve essere aggiunta una nuova API ai microservizi attuali, questa API ha il compito di autenticare l'utente che richiede il servizio di latenza, controllare se ha i permessi necessari per eseguire l'operazione, interrogare il servizio di upstream, ricevere la risposta e infine recuperare le informazioni necessarie per mostrare i risultati all'interno della mappa.

Sono necessarie le seguenti informazioni aggiuntive:
- Le geolocalizzazione (latitudine e longitudine a livello di città, nome città e nazione) di ciascun server interrogato.
- La geolocalizzazione dell'indirizzo (latitudine e logitudine a livello di nazione) ip dato dal client al momento della richiesta.

L'integrazione con l'applicativo dovrà essere coerente con la codebase già esistente.
## Progettazione
In questo capitolo verranno speigate in maniera dettagliata le decisioni architetturali prese per lo sviluppo del servizio.
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
Il numero di server da interrogare potrebbe aumentare notevolmente nel tempo; Eseguire le chiamate in sequenza non è possibile, in quanto con un numero elevato di interrogazioni da condurre l'esecuzione verrebbe sicuramente interrotta dal timeout imposto.
È necessario quindi eseguire le chiamate in parallelo.
L'operazione è di tipo *embarassingly parallel* in quanto ogni singola esecuzione è indipendente dalle altre.
Senza dare alcuna limitazione il sistema potrebbe consumare una elevata quantità di risorse in quanto il numero di server da interrogare potrebbe aumentare notevolmente. Ciò violerebbe il vincolo posto in precedenza (capitolo analisi).
È stato scelto di limitare il numero di unità di esecuzione sfruttando il _bounded concurrency pattern_, che attraverso un meccanismo di semafori, la concorrenza massima viene regolata per evitare il degrado della rete e limitare l'utilizzo eccessivo di risorse della macchina.

### Gestione dei middleware
A seguito di una richista http potrebbe essere necessario eseguire multiple funzioni con scopo diverso.
Si vuole poter eseguire le funzioni in sequenza, avendo la possibilità di interrompere l'esecuzione in qualsiasi momento.

Il sistema per la gestione dei middleware in sequenza utilizza il design pattern *chain of responsibility* che permette di passare la richiesta lungo una catena di "handler".
Al ricevimento di una richiesta dopo averla processata, un handler può decidere se passarla al prossimo handler all'interno della catena o se interrompere l'esecuzione.

Alcuni esempi di handler possono essere: la gestione del logging strutturato e il disaster recovery.

Sia il servizio di downstream che di upstream utilizzeranno questa struttura.
<!-- #### Caricamento delle variabili di configurazione
È necessario poter caricare delle variabili di ambiente come configurazione del servizio.
Ciascun servizio ha una serie di variabili obbligatorie e opzionali.
Una variabile obbligatoria deve essere presente all'avvio del servizio, la mancata presenza deve causare una chiusura istantanea del servizio; Le variabili opzionali possono non essere presenti durante l'inizializzazione, in questo caso verranno decisi dei valori di default.

Viene definita una classe con una serie di metodi per fare il parsing del valore delle variabili d'ambiente;

Il fallimento di una di queste funzioni, dato da un valore non coerente con il tipo richiesto o la mancata presenza di una variabile obbligatoria risulta nell'immediata terminazione del programma. -->
### Integrazione con FlashStart 2026
Si vuole permettere agli utilizzatori di *FlashStart 2026* di usufruire del servizio di latenza.
È dunque necessario aggiungere un entrypoint all'interno della nuova applicazione.
Per mantenere una codebase coerente con il resto dell'applicativo sviluppato dagli altri developer in azienda, è stato scelto di progettare il nuovo entrypoint con il pattern `M.V.C`.

Si divide l'entrypoint in 4 sezioni:
1. Un `LatencyController`, classe intermedia tra la view e il servizio.
2. Il `LatencyService`, contenente l'effettiva business logic e il controllo dei permessi.
3. Una serie di Repositories che fungono da tramite tra il servizio e le fonti di dati come un database, per `GeolocalizationRepository` e `ServerDataRepository`, o come il servizio upstream per `LatencyRepository`.

Al fine di mantenere separata la logica del servizio con l'autenticazione e autorizzazione, questi ultimi verranno controllati per tramite del pattern *decorator*, sfruttando il codice già presente.

Questo sistema permette una elevata semplicità di manutenzione grazie alla accurata divisione dei concetti.
## Sviluppo
Durante questa fase sono stati analizzati i possibili linguaggi da utilizzare per la ricostruzione del servizio.
Per rispettare i requisiti del problema (servizio a basso consumo di risorse), il linguaggio scelto deve essere un linguaggio compilato e non interpretato; Linguaggi come python e php sono stati scartati a priori.

Il secondo requisito è la possibilità di gestire codice in maniera parallela.
Dati questi due principali requisiti è stato scelto `go` come linguaggio di scrittura dei servizi di upstream e downstream, poiché è un linguaggio compilato e soprattutto supporta in maniera nativa e facilitata la gestione di routine parallele e fornisce inoltre una struttura dati apposita (canale o `chan`) per la gestione concorrente dei dati, concetto fondamentale per lo sviluppo degli strumenti richiesti.
### Implementation Highlight
#### Wrapper ICMP
L'esecuzione del ping sul server è l'operazione più importante del sistema.
La versione legacy del software eseguiva semplicemente il comando `ping -c MAX_PING <network>` eseguendo poi una normalizzazione del risultato. Come detto in precedenza il parser si basa sulla manipolazione di una stringa con indici di riga fissi; Questo rende il sistema molto fragile ad errori, una piccola variazione nel formato del risultato del comando `ping` potrebbe causare il fallimento della richiesta.

Il nuovo sistema è stato creato con l'affidabilità alla base.
Al posto di eseguire una shell bash, è stata utilizzata la libreria di `golang.org/x/net/icmp` per eseguire chiamate `ICMP` attraverso la rete; A questa libreria è stato creato un wrapper che permetta l'utilizzo facilitato di una libreria altrimenti altamente complessa.

Il wrapper permette di eseguire una serie di messaggi `ICMP:Echo` con id univoci in modo da filtrare pacchetti `ICMP` non inviato dal sistema, attraverso il pattern strategy definito in precedenza, permette di inviare in maniera trasparente la richiesta sia su una rete IPv4 che su una rete IPv6, in base all'indirizzo fornito dall'utente. Infine permette opzionalmente la possibilità di impostare il `TTL` ad ogni pacchetto.

``` go
func (con *EchoConnection) SendEchoWithTTL(content string, ttl int) (*EchoResponse, error) {
	// save request id to filter it later
	requestID := rand.Intn(RANDOM_ID_MAX)

	// Build ICMP echo request
	msg := icmp.Message{
		Type: con.ipVersionStrategy.GetIcmpEchoType(),
		Code: 0,
		Body: &icmp.Echo{
			ID:   requestID,
			Seq:  con.sequenceNumber,
			Data: []byte(content),
		},
	}

	con.ipVersionStrategy.SetTTL(ttl)

	b, err := msg.Marshal(nil)
    // ... error check

    // sends the message and waits its answer
	start := time.Now()
	if _, err := con.connection.WriteTo(b, con.address); err != nil {
		err = errors.New("Connection unavailable")
		slog.Error(err.Error())
		return nil, err
	}

	// Wait for a reply
	con.connection.SetReadDeadline(time.Now().Add(timeout))
	for {
		n, peer, err := con.connection.ReadFrom(reply)
		rtt := time.Since(start)

		if err != nil {
			// ... Timeout — no response
		}

		// Parse the ICMP reply
		parsedReply, err := icmp.ParseMessage(reply)
		// ... error check
		responseType := con.ipVersionStrategy.CheckResponseType(parsedReply.Type)
		// If its not the package sent before, continue listening
		// fmt.Printf("Response: %s from addr: %s\n", parsedReply.Type, peer.String())
		switch responseType {
		case networkstrategy.ICMPEchoReply:
            // handle echo reply
		case networkstrategy.ICMPTimeExceeded:
            // handle time exceeded error
		default:
		}

		response := NewEchoResponse(/* response data */)
		return response, nil
	}
}
```
Quest'ultima feature permette una potenziale futura espansione del servizio attuale con l'aggiunta del traceroute.

Questo wrapper risolve il problema della fragilità del risultato, poiché il parsing del risultato non viene più fatto in base all'output di un comando di una bash. Il risultato finale viene calcolato sulla base dei risultati di ogni pacchetto che vengono salvati in locale.

#### Bounded Concurrency
Il linguaggio `golang` mette a disposizione due costrutti nativi per la gestione di routine concorrenti: le goroutine e i canali (`chan`) strutture dati in grado di gestire automaticamente la concorrenza senza provocare race conditions.
I canali possono anche essere visti come dei semafori nella teoria della programmazione concorrente.

L'implementazione del bounded concurrency pattern si è basata da una funzione che calcola il *checksum MD5* di tutti i file in una directory specificata, riadattandola secondo le necessità del sistema in sviluppo.

``` bibtex
@misc{ajmani2014pipelines,
  author       = {Ajmani, Sameer},
  title        = {Go Concurrency Patterns: Pipelines and Cancellation},
  howpublished = {The Go Blog},
  year         = {2014},
  month        = {March},
  url          = {https://go.dev/blog/pipelines},
  note         = {Licensed under CC BY 4.0.}
}
```

Il funzionamento si basa su 3 canali distinti:
- **Canale di distribuzione degli indirizzi**: eroga uno alla volta ciascun indirizzo dei server da interrogare. Su questo canale si metteranno in ascolto i lavoratori.
- **Canale di raccolta dei risultati**: eseguito dal thread principale colleziona i risultati calcolati da ciascun thread lavoratore.
- Un canale che segnala la conclusione.

Una volta inizializzati i canali viene creato l'insieme di lavoratori che eseguiranno le richieste di latenza ai vari server.
A ciascun worker vengono condivisi i tre canali creati, quando riceve un nuovo indirizzo IP dal canale di distribuzione, esegue la chiamata, ritorna il valore di latenza e si rimette in ascolto per il prossimo indirizzo.

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

Questo permette una effettiva parallelismo a runtime solamente se il sistema sottostante ha una `CPU` multicore e se la variabile `GOMAXPROCS` lo permette.
- `GOMAXPROCS` è la variabile che indica il numero massimo di thread che si possono usare per eseguire goroutines in parallelo.

Dalla versione 1.24 in poi la variabile assume di default il numero totale di CPU cores nella macchina.
``` bibtex
@misc{ajmani2014pipelines,
  author       = {Michael Pratt, Carlos Amedee},
  title        = {Container-aware GOMAXPROCS},
  howpublished = {The Go Blog},
  year         = {2025},
  month        = {August},
  url          = {https://go.dev/blog/container-aware-gomaxprocs},
  note         = {Licensed under CC BY 4.0.}
}
```

#### Integrazione con FlashStart 2026
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
Dopo una analisi della semantica delle informazioni fornite da questo servizio, insieme al team di sviluppo si è giunti alla conclusione che nessun microservizio attualmente presente sarebbe stato adatto al contenimento di questa nuova API, è stata quindi necessaria la creazione di un nuovo microservizio all'interno di *FlashStart 2026*: `infrastructure`.

Il nuovo microservizio servirà per tutte quelle nuove feature che implementano un qualsiasi tipo di strumento di supporto all'utilizzo dell'applicazione.
Per la creazione del nuovo microservizio è stato necessario configurare diversi aspetti:
- Ambiente di testing dei sorgenti del nuovo microservizio.
- Nuovo database con un collegamento diretto.
- Due dockerfile, uno per l'ambiente di development e testing e uno per l'ambiente di esercizio (staging), necessario per poi dispiegare il nuovo microservizio all'interno di un cluster kuberneetes.

### Backend
Per il dispiegamento dei servizi di downstream e upstream all'interno della infrastruttura esistente sono stati creati dei dockerfile ad hoc contenenti i file binari compilati dei due servizi.
Per evitare di caricare sui server codice binario inutilizzato (come il compilatore del linguaggio), è stato creato un dockerfile multistage
- Un primo stage che scarica le librerie necessarie e compila i codici sorgenti in un unico file binaro eseguibile.
- Un secondo stage che utilizzando una immagine leggera (`FROM debian:bookworm-slim`) elimina qualsiasi file non necessario e successivamente copia il file binario creato al primo stage, minimizzando le risorse necessarie per l'esecuzione del servizio.

A differenza del vecciho downstream, dispiegato su ciascun DNS resolver (multipli container in una sola macchina), il nuovo servizio verrà direttamente inserito all'interno sistema bare metal come servizio indipendente.

Questa modifica ha come conseguenza due miglioramenti: minore consumo di memoria in una singola macchina dato da inutili copie del servizio e maggiore efficienza nel processo di aggregazione, in quanto il numero di server da interrogare è notevolmente ridotto, viene inoltre rimossa la necessità di filtrare indirizzi duplicati in quanto ogni macchina viene interrogata una volta sola.
## Analisi dei Risultati
### Testing automatico

## Conclusioni
L'esperienza di tirocinio svolta ha permesso di completare con successo l'intero ciclo di vita del software per la modernizzazione dei servizi di latenza. Il passaggio dall'architettura PHP/Python al nuovo ecosistema in Go ha portato notevoli benefici prestazionali e strutturali.
Efficienza Computazionale: Il passaggio a binari nativi compilati in contenitori Debian-Slim ha ridotto notevolmente l'utilizzo della memoria RAM rispetto al sistema basato su Apache, PHP e Python.
Robustezza Operativa: La gestione manuale delle chiamate ping garantiscono una alta affidabilità rispetto al sistema preesistente.
Trade-off velocità/affidabilità: L'aggiunta di diversi layer aggiuntivi rispetto all'esecuzione nativa del ping del sistema operativo riporta dei leggeri rallentamenti nell'esecuzione del ping; Rispetto all'esecuzione nativa, il nuovo applicativo riporta leggeri rallentamenti ($50-70 \mu s$).
Questo rallentamento è però trascurabile, poiché la latenza richiede una precisione nelle misurazioni nell'ordine dei millisecondi. Inoltre questo rallentamento è visibile solo nel caso specifico che venga eseguito il ping sull'interfaccia di loop-back, una richiesta possibile ma che ha poco senso ai fini dello strumento.

Documentazione: la scrittura della documentazione in parallelo con lo sviluppo del codice ha permesso una dettagliata descrizione di tutte le componenti del sistema che facilitano eventuali modifiche e aggiunte.
### Lavori futuri
Traceroute