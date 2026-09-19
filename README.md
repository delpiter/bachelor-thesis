# Benchmarking Latency in a Global Anycast Network

> Bachelor's thesis in Computer Science and Engineering, University of Bologna, Cesena Campus, developed in collaboration with [FlashStart](https://flashstart.com/). Supervisor: Prof. Mirko Viroli.

## Abstract

This project focuses on the architectural modernization and re-engineering of latency measurement services within an anycast network. The pre-existing system, based on a PHP/Apache architecture with Python fan-out scripts, suffered from performance limitations, high CPU/memory overhead, low response reliability, and excessive rigidity in handling concurrent I/O.

The project involved rewriting two core sub-services. The downstream service, responsible for measuring the latency between the server itself and the target machine specified as input, was originally written in PHP and was redesigned in Go, adopting Clean Architecture principles and a design-pattern-based approach to maximize response reliability. The upstream service, responsible for aggregating the data collected from downstream server queries, was originally written in Python and was also rewritten in Go, leveraging the language's native concurrency model (goroutines and channels) to overcome the concurrency I/O rigidity found in the legacy system. Finally, the integration code connecting the backend services to the company platform was developed.

The new system was validated against the pre-existing architecture, showing improved service reliability and reduced memory usage on the deployment servers. The work also included the design of a modernized user interface for the new application.

---

Fammi sapere se vuoi che aggiunga sezioni tipiche di un README di repo LaTeX (es. struttura cartelle, come compilare la tesi, licenza, ringraziamenti).