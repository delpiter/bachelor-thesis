# Servizio di diagnotica della latenza in una rete anycast

## Introduzione

## Background
> Struttura attuale del servizio

- Come funziona il servizio
- Modalità di interazione tra i vari servizi.
- Grafici cool (flusso dell'informazione e schema delle macchine).
## Analisi
In questo capitolo vengono evidenziati i requisiti del progetto, analizzando i limiti imposti dalla quantità di risorse a disposizione da ciascun server all'interno della rete.
### Obbiettivi
L'obbiettivo di questo progetto è quello di riscrivere un servizio di monitoraggio della latenza, partendo da un codice monolitico scritto in php e javascript.
La riscrittura coinvolge 4 sezioni distinte:
- Servizio downstream
    - Servizio localizzato all'interno di ciascuno dei server remoti nella rete utilizzato per richiedere il tempo di latenza tra il server e una macchina specifica in una qualsiasi location nel mondo.  
- Servizio upstream
    - Servizio localizzato all'interno del server principale, questo servizio ha il compito di recuperare la lista dei server attivi da interrogare per la latenza, aggregare e riordinare i risultati ottenuti.
- Frontend
    - Il frontend servirà per visualizzare i risultati ottenuti in maniera semplice e interpretabile anche da un utilizzatore non esperto del software.
### Limitazioni
Per quanto riguarda i servizi di downstream e upstream le limitazioni sono le seguenti:
- Il software deve essere il più leggero e efficiente possibile, in quanto dovrà essere dispiegato su server che devono gestire milioni di richieste giornaliere.
- Il software non deve assolutamente rallentare il funzionamento delle macchine fisiche.
- Il nuovo software deve essere retrocompatibile con la versione precedente del software, in modo tale che possa essere utilizzato dalla dashboard legacy senza necessità di ulteriori modifiche al codice attuale.
### Analisi dei Requisiti
I due servizi principali sono stati progettati seguendo il pattern di programmazione MVC.
- Devono esporre delle api ben definite (View), analizzare le richieste ricevute (Controller) e eseguire le operazioni (Controller).
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
La richiesta si divide in due sezioni distinte
> Recupero della lista dei client
In base alla richiesta ricevuta viene selezionata una lista diversa di server da interrogare.
- `api/latency/` seleziona gli host standard da interrogare.
- `api/latency/cloud` selezione una lista di host sul cloud.

Il fallimento della richiesta o il superamento di una deadline comporta il fallimento della richiesta di latenza.

> Aggregazione dei valori
In questa sezione viene validato ciascun indirizzo ip ottenuto in precedenza; Per ogni host valido viene interrogato il server server downstream corrispondente per il valore della latenza.
Ogni richiesta deve essere trattata in maniera indipendente dalle altre: il fallimento di una richiesta non deve comportare l'intera aggregazione.
Successivamente vengono normalizzati i risultati ottenuti per aderire al modello di risposta analizzato in precedenza.
- Se una richiesta fallisce, impiega troppo tempo o ritorna un codice http diversa da `2xx`, comporta l'assegnazione del valore `-1` alla richiesta a quel server.

Infine la lista di valori deve essere ordinata in ordine ascendente secondo la latenza.
#### Downstream service
Il servizio di downstream comprende 3 sezioni:
- Esposizione dell'interfaccia API per la richiesta della latenza.
`GET /api/latency`
Appena ricevuta la richiesta, essa deve essere validata.
La richiesta dovrà ritornare un semplice valore che rappresenta il tempo di risposta medio del server interrogato.
Una richiesta valida deve contenere un parametro `network` nella richiesta `http` e come valore deve essere presente un indirizzo ip valido sia ipv4 che ipv6.
- Se la richiesta non soddisfa i requisiti deve fallire immediatamente per evitare di sprecare risorse importanti del server.

Una volta validato l'indirizzo viene eseguito il ping della macchina specificata, questo passaggio deve essere indipendente dal tipo di versione dell'indirizzo che è stato ricevuto.
- Questa operazione ha delle deadline strette per evitare di consumare troppe risorse, se una richiesta ping eccede questa deadline il pacchetto viene considerato perso.
- Se in un qualsiasi momento accade un errore, la richiesta non deve fallire, invece si deve attivare un meccanismo di fallback e ritornare un valore di default (`-1`).

Infine deve essere fatta la validazione del risultato ottenuto dall'esecuzione del ping.
- Se la perdita di pacchetti blocca il calcolo del tempo di risposta medio, la richiesta deve tornare `-1`.

## Progettazione
In questo capitolo verranno speigate in maniera dettagliata le decisioni architetturali prese per lo sviluppo dei due serivzi principali.
### Downstream Service
Il servizio di downstream deve essere in grado di gestire in maniera trasparente l'utilizzo di versioni del protocollo IP differenti (`ipv4` e `ipv6`).
Le operazioni specifiche ad una versione del protocollo `ip` verranno delegate ad una classe che, attraverso l'utilizzo del pattern *strategy*, sarà in grado di gestire l'operazione in base alla verisone del protocollo ip utilizzata.
### Upstream Service
Il numero di server da interrogare potrebbe aumentare notevolmente nel tempo; Eseguire le chiamate in sequenza non è possibile, in quanto con un numero elevato di interrogazioni da fare l'esecuzione verrebbe sicuramente interrotta dal timeout imposto.
È necessario quindi eseguire le chiamate in parallelo.
L'operazione è di tipo *embarassingly parallel* in quanto ogni singola esecuzione è indipendente dalle altre, sarebbe dunque facilmente parallelizzabile assegnando un thread a ciascuna richiesta da fare.
Questo però andrebbe a consumare una elevata quantità di risorse, e ciò violerebbe il vincolo posto in precedenza (capitolo analisi).
È stato scelto di limitare il numero di thread lavoratori sfruttando il _bounded concurrency pattern_.
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
## Sviluppo
### Implementation Highlight

## Deployment

## Analisi dei Risultati
### Testing automatico
### Sperimentazioni

## Conclusioni
### Lavori futuri

## Bibliografia