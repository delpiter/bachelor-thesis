# Frontespizio
Benchmarking delle latenze in una rete Anycast globale.
- Foschi Gioele
- Mirko Virolo
- Francesco Collini
- Stefano Babini

Tesi in Azienda
# Contesto
- FlshStart è una azienda che da oltre 20 anni protegge gli utenti nel mondo
digitale con soluzioni di sicurezza informatica attraverso il filtraggio dei contenuti basato su DNS.
- Opera su una rete anycast globale composta da 160+ nodi distribuiti in tutti i continenti.
- Dal 2026 ha iniziato il redesign della piattaforma.
- Uno dei componenti di cui fare il redesign è il servizio di controllo latenza.

## Latency Tool
- Lo strumento di latenza consiste nella visualizzazione delle latenze di ciascun server nella rete anycast rispetto ad un indirizzo specificato.
- Strumento fondamentale per il network admin.
    - Stima la latenza di risposta del DNS;
    - Ed eventuali nodi alternativi in caso di problematiche.

# Obbiettivi
- Migrazione del servizio da PhP/Python, verso una nuova piattaforma progettata secondo criteri più moderni e strutturati.
    - Modernizzazione architetturale e sul re-engineering delle tre componenti del servizio: upstream, downstream e l'integrazione del servizio con la nuova piattaforma.
- Mantenimento delle firme dei metodi per garantire la retrocompatibilità.
- Ricostruzione dell'interfaccia grafica seguendo lo stile della nuova piattaforma.

## Linguaggio utilizzato
Go, motivazioni
- Compilazione nativa
- Binari statici
- Concorrenza nativa
- Gestione automatica della memoria
- Libreria standard e semplicità

## Upstream Aggregator
Componente centrale del servizio
- Recupero dinamico della lista di server da interrogare.
- Invio delle richieste di latenza ai vari indirizzi ottenuti.
- Aggregazione dei risultati.

Sviluppato con un paradigma di programmazione parallela master-worker, limitando il numero di worker disponibili (bounded parallelism)
per evitare il degrado della rete per via delle troppe chiamate http e l'eccessivo consumo di risorse del server se numerosi 
server fossero interrogati nello stesso momento.
## Downstream
Componente responsabile del calcolo della latenza dal server in cui è distribuito all'indirizzo specificato sulla richiesta
- Espone un singolo endpoint
- Wrapper ICMP: Sviluppato con librerie per l'invio di messaggi ICMP 
- Gestione trasparente delle versioni IP con il pattern strategy
## Integrazione con FlashStart
Componente sostitutiva alla route php nell'applicativo legacy
- Sviluppato con il pattern MVC per mantenere coerenza con il resto dell'applicativo
- Un controller: elemento intermediario tra la view e il servizio vero e proprio
- Un Service: contenente l'effettiva business logic e il controllo dei permessi
- Più repository: una serie di repository, che fungono da tramite tra il servizio e le fonti di informazione
### Microservizio Infrastructure
Nessun microservizio attualmente presente all'interno della nuova piattaforma sarebbe stato adatto al contenimento del servizio latency.
- Creazione del nuovo microservizio "infrastructure", usato per qualsiasi strumento di supporto all'utilizzo dell'applicazione.
- Ambiente di Testing: automatico tramite l'utilizzo di librerie come Mocha, Chai e Sinon
- Database: un database PostgreSQL specifico al microservizio
- Dockerfile

### Frontend
Il design dell'interfaccia ha attraversato 3 fasi evolutive:
- Iniziale: analisi della struttura legacy e creazione del wireframe.
- Prima versione: semplice modernizazione dell'interfaccia.
- Versione finale: sviluppata in collaborazione con l'esperto di UX aziendale, disegnata mantenendo coerente lo stile visivo con il resto della piattaforma.
# Conclusioni
- Efficienza Computazionale
- Documentazione
- Robustezza operativa
## Lavori Futuri
Possibile estensione futura riguarda l'implementazione di un meccanismo di traceroute applicativo, pensato per tracciare il percorso di rete attraversato dalle richieste dell'utente.
Il sistema attuale è stato progettato secondo criteri di modularità ed estendibilità.
l'introduzione del traceroute sul server di downstream non richiederebbe una riprogettazione del sistema, ma si limiterebbe all'aggiunta di una nuova classe che sfrutta il wrapper ICMP già progettato e integrato nell'architettura esistente