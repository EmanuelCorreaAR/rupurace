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
go install github.com/EmanuelCorreaAR/rupurace/cmd/rupurace@v0.4.0
```


## Quick start

El corazón del proyecto:

```bash
# encuentra un bug y guarda el witness
go run ./cmd/rupurace explore -o witness.json

# reduce el contraejemplo a 1-minimal (mantiene el original)
go run ./cmd/rupurace explore --minimize -o witness.json
# → witness.json (evidencia) + witness.min.json (explicación)

# o a partir de un witness ya guardado
go run ./cmd/rupurace minimize witness.json

# Reproduce el mismo fallo, siempre
go run ./cmd/rupurace replay witness.json
go run ./cmd/rupurace replay witness.min.json
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
| `interleave` | sustrato sintético W×S (medir explosión) | (ninguna) |

```bash
go run ./cmd/rupurace explore --scenario check-then-act -o witness.json
go run ./cmd/rupurace replay witness.json

# medir el espacio antes de podar
go run ./cmd/rupurace explore --scenario interleave --workers 2 --steps 5 --stats

# podar nodos de exploración equivalentes (estado + enabled)
go run ./cmd/rupurace explore --scenario interleave --workers 2 --steps 5 --stats --prune
```


## Medir la explosión

Antes de partial-order reduction, RupuRace mide — y opcionalmente pode:

```text
# sin --prune (baseline 0.2.0)
Schedules considered: 252
Schedules executed:   252
States visited:       923
Unique states:        36
Equivalent pruned:    0

# con --prune (0.2.1)
Schedules considered: 252
Schedules executed:   <252
States visited:       <923
Unique states:        36
Equivalent pruned:    >0
```

`UniqueStates` es medición. `EquivalentPruned` es optimización realmente aplicada.

La equivalencia **no** es solo el estado de negocio: el nodo de exploración es
`fingerprint(state) + enabled transitions`, para no confundir futuros distintos
con el mismo dato compartido.

Con `--prune`, prefijos que conmutan al mismo nodo (p.ej. `A0 B0` ≡ `B0 A0`)
se exploran una sola vez: puede bajar el conteo de schedules/violaciones
*por camino*, pero el primer witness determinista y el bug siguen alcanzables.

`--continue` sigue explorando después del primer fallo (cuenta Violations).


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

### Minimización (0.3.0)

RupuRace distingue evidencia de explicación:

| Artefacto | Rol |
|-----------|-----|
| `witness.json` | Evidencia cruda del explorador (`Woriginal`) |
| `witness.min.json` | Contraejemplo 1-minimal derivado (`Wminimal`) |

`Minimize` promete: no se puede quitar una transición más sin dejar de
reproducir **la misma violación semántica** (mismo invariante). `failedAt`
puede cambiar. No promete todavía el shortest-global.

```bash
go run ./cmd/rupurace explore --scenario lost-update --minimize -o witness.json
```


## Commands

| Command | Rol |
|---------|-----|
| `explore` | Explora schedules; con `-o` escribe witness al fallar; `--minimize` también escribe `*.min.json` |
| `replay` | Reejecuta un witness de forma determinista |
| `minimize` | Deriva un witness 1-minimal a partir de uno existente |
| `version` | Muestra la versión |


## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Usage or I/O error |
| `2` | Invariant violation with `--fail-on-violation` |


## Estado

**0.4.0** — Puente mínimo a Go real: `internal/gate` (`Await` + Quiesce) ejecuta
goroutines reales y entra al pipeline consolidado Explore → `witness_v1` →
Minimize → Replay. Demo: check-then-act en vivo. Sin API pública todavía.

**Spike:** `spike/synctest/NOTES.md` — veredicto B (synctest ≠ Explore).

**Next:** pulir semántica del punto de control; más demos; eventualmente CLI.
Sin reescritura obligatoria del SUT al modelo abstracto `{worker,step}` tables.


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
