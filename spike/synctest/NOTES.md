# Spike: testing/synctest × RupuRace

Fecha: 2026-09-23  
Go: 1.27.1 (`testing/synctest` estable desde 1.25)  
Estado: descartable — no es API de producto ni 0.4.0

## Pregunta

¿Hasta dónde podemos observar/controlar código Go real con `synctest`
sin obligar al usuario a reescribirlo como `{worker, step}`?

## Resultados empíricos

| Experimento | Resultado |
|-------------|-----------|
| Reloj falso + `time.Sleep` | ✓ Avanza solo con bloqueo durable |
| `synctest.Wait` | ✓ Espera quiescencia durable |
| Elegir / enumerar interleavings de 2 Gs *runnable* | ✗ Sin API; a veces hay ruido de orden, no exploración |
| `sync.Mutex` + `Wait` | ✗ Lock **no** es durable; sin `Unlock` el bubble no asienta |
| Gates por canal + Wait (híbrido) | ✓ Se fuerza check-then-act bueno y malo a voluntad |

## Veredicto: **B**

`synctest` ayuda con **tiempo / quiescencia / aislamiento de bubble**,
pero **no** sustituye un explorador de schedules.

No hay API para:

- listar goroutines runnable
- elegir cuál corre a continuación
- generar sistemáticamente el espacio de interleavings

Observar A-primero y B-primero entre bubbles distintos no es Explore:
es ruido de scheduling sin control ni testimonio reproducible por diseño.

`sync.Mutex` queda fuera del modelo durable: no sirve como sonda
de scheduling vía `Wait` sola.

## Implicación para RupuRace

El puente a Go real **no** es “meter el código en un bubble y Explore gratis”.

El puente viable hoy:

```text
código Go
   │  puntos de control explícitos
   │  (canales / hooks / instrumentación)
   ▼
schedule cooperativo  ←── mismo contrato que hoy
   ▼
Explore → Witness → Minimize → Replay
   │
   └─ synctest como harness (Wait / fake clock)
```

`synctest` encaja como **infraestructura parcial** del harness:

- asentar el sistema tras cada transición (`Wait`)
- eliminar flakiness de wall-clock en demos/tests
- aislar goroutines del SUT en un bubble

No encaja como reemplazo del scheduler de RupuRace.

## Outcomes del plan original

| | Significado | ¿Encaja? |
|-|-------------|----------|
| A | synctest da control suficiente de scheduling | no |
| **B** | ayuda tiempo/goroutines; no scheduling libre | **sí** |
| C | no sirve para exploración | parcial — sí como harness, no como Explore |

## Next (cuando se abra 0.4.x)

1. Diseñar la forma mínima de **gate/hook** sobre Go real (sin API pública todavía).
2. Usar `synctest` solo en el harness de tests del puente.
3. Seguir emitiendo `witness_v1` + `*.min.json` — el destino del pipeline no cambia.

## Seguimiento (0.4.0)

Implementado en `internal/gate`:

- `Controller.Await` / `Release` / `Quiesce` / `Go`
- `Explore` (mundo fresco por candidato)
- `Replay` + `ReplayScenario` → Minimize
- Demo live `CheckThenAct`

```bash
go test ./internal/gate/ -v
```

## Cómo correr

```bash
go test ./spike/synctest/ -v
```
