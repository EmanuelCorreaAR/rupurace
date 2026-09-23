# RupuRace

**Explore schedules. Find invariant violations. Keep the witness.**

Parte de la familia **Rupu**.

Systematic schedule exploration for Go programs with deterministic failure replay.

RupuRace explora schedules de ejecución sobre escenarios **cooperativos**
(explícitos) para exponer fallos dependientes del orden y convertirlos en
witnesses reproducibles.

```text
Scenario
   ↓
Possible transitions
   ↓
Schedule explorer
   ↓
State transition
   ↓
Invariant check
   │
   ├── OK → continue exploring
   │
   └── FAIL
          ↓
       Witness
          ↓
       Replay
```

No controla goroutines arbitrarias del runtime de Go. Controla pasos
explícitos por worker (`A0`, `A1`, `B0`, …) y elige el orden.


## Por qué

Los bugs de concurrencia son difíciles de reproducir: la ejecución que
disparó el fallo puede no volver a ocurrir nunca.

RupuRace toma otro enfoque:

1. Explorar schedules de forma deliberada.
2. Chequear invariantes después de cada transición.
3. Capturar el schedule que causó el fallo.
4. Reproducir ese fallo exacto, siempre igual.

El objetivo no es solo encontrar un fallo.

El objetivo es conservar la evidencia necesaria para reproducirlo.


## Install

Requiere Go 1.22+.

```bash
git clone https://github.com/EmanuelCorreaAR/rupurace.git
cd rupurace
go test ./...
go run ./cmd/rupurace --help
```

Cuando haya tags de release:

```bash
go install github.com/EmanuelCorreaAR/rupurace/cmd/rupurace@v0.1.2
```


## Quick start

El corazón del proyecto:

```bash
# encuentra un bug y guarda el witness
go run ./cmd/rupurace explore -o witness.json

# Reproduce el mismo fallo, siempre
go run ./cmd/rupurace replay witness.json
```

Scenario default: `double-withdraw` — 2 workers × 3 steps, balance compartido,
invariante `balance >= 0`, exploración **exhaustiva**.

```bash
go run ./cmd/rupurace explore --json
go run ./cmd/rupurace explore --fail-on-violation   # exit 2 si hay violación
go run ./cmd/rupurace explore --scenario lost-update -o witness.json
```


## Scenarios

La abstracción `{worker, step} → state → invariant → witness` se prueba contra
varios problemas distintos (motor en `internal/`, sin API pública todavía):

| Scenario | Bug clásico | Invariante |
|----------|-------------|------------|
| `double-withdraw` | withdraw race / balance negativo | `balance >= 0` |
| `lost-update` | read-modify-write perdido | `value == 2` |
| `check-then-act` | TOCTOU al reclamar un recurso | `owners <= 1` |
| `init-ordering` | publish antes de escribir el payload | `got == 0 \|\| got == expected` |

```bash
go run ./cmd/rupurace explore --scenario check-then-act -o witness.json
go run ./cmd/rupurace replay witness.json
```


## Witness

```json
{
  "scenario": "double-withdraw",
  "schedule": [
    {"worker": "A", "step": 0},
    {"worker": "B", "step": 0},
    {"worker": "A", "step": 1},
    {"worker": "B", "step": 1},
    {"worker": "A", "step": 2},
    {"worker": "B", "step": 2}
  ],
  "invariant": "balance >= 0",
  "failedAt": 6
}
```

`rupurace replay witness.json` falla en la misma transición, cada vez.


## Commands

| Command | Rol |
|---------|-----|
| `explore` | Explora schedules; con `-o` escribe witness al fallar |
| `replay` | Reejecuta un witness de forma determinista |
| `version` | Muestra la versión |


## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Usage or I/O error |
| `2` | Invariant violation with `--fail-on-violation` |


## Estado

**0.1.2** — Propiedad constitucional testeada en `replay`; round-trip `witness_v1`
como producto. Battery de scenarios en `0.1.1`. Motor en `internal/`. Sin API Go pública.

**Next:** bounded exploration / state hashing cuando el espacio crezca; después
witness minimization. Instrumentar Go concurrente real viene más tarde.


## Principios

- Mismo scenario + misma configuración → mismo resultado
- Transiciones cooperativas explícitas (no el scheduler de Go)
- Sin dependencia del wall-clock
- Los fallos producen evidencia inspeccionable
- La reproducción es una feature de primer nivel
- Local-first
- Amigable con CI


## Qué no es

- **No es un reemplazo de `go test -race`**

| | Pregunta |
|-|----------|
| `go test -race` | ¿Ocurrieron accesos de memoria conflictivos? |
| **RupuRace** | ¿Existe un orden de estas operaciones que viole una propiedad del sistema? Si sí → mostrame cuál, guardalo, reproducilo. |

- No controla goroutines / mutexes / channels del runtime (aún)
- No es un scheduler de producción
- No es una plataforma de observabilidad
- No es distributed tracing
- No es una herramienta de debugging con IA


## Apoyar el proyecto

Si RupuRace te sirve, podés invitarme un cafecito: [cafecito.app/emacorreadev](https://cafecito.app/emacorreadev)


## License

Apache License 2.0
