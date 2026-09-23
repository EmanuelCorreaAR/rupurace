# RupuRace

**Explore schedules. Find invariant violations. Keep the witness.**

Parte de la familia **Rupu**.

Testing de concurrencia determinista para Go.

RupuRace explora schedules de ejecución alternativos para exponer
fallos dependientes del orden y convertirlos en witnesses reproducibles.

```text
concurrent execution
        ↓
schedule exploration
        ↓
invariant violation
        ↓
reproducible witness
        ↓
deterministic replay
```


## Por qué

Los bugs de concurrencia son difíciles de reproducir: la ejecución que
disparó el fallo puede no volver a ocurrir nunca.

RupuRace toma otro enfoque:

1. Explorar schedules de ejecución de forma deliberada.
2. Chequear invariantes después de cada transición de estado.
3. Capturar el schedule que causó el fallo.
4. Reproducir ese fallo exacto de forma determinista.

El objetivo no es solo encontrar un fallo.

El objetivo es conservar la evidencia necesaria para reproducirlo.


## Estado

Desarrollo temprano.

Milestone inicial:

- scheduler determinista
- ejecución de escenarios
- chequeo de invariantes
- witnesses de fallo
- replay determinista

El motor vive bajo `internal/`. Todavía no hay API Go pública ni comandos
de CLI — ambos aparecen solo cuando existan y estén testeados.

Module path: `github.com/EmanuelCorreaAR/rupurace`


## Principios

- Mismo scenario + misma configuración → mismo resultado
- Sin dependencia del wall-clock
- Sin dependencia del scheduler de Go
- Los fallos producen evidencia inspeccionable
- La reproducción es una feature de primer nivel
- Local-first
- Amigable con CI


## Qué no es

- **No es un reemplazo de `go test -race`**
- No es un scheduler de producción
- No es una plataforma de observabilidad
- No es distributed tracing
- No es una herramienta de debugging con IA


## Apoyar el proyecto

Si RupuRace te sirve, podés invitarme un cafecito: [cafecito.app/emacorreadev](https://cafecito.app/emacorreadev)


## License

Apache License 2.0
