# Gate abstraction limits (0.4.1)

Stressing `Await` / `Release` / `Quiesce` the way NodeKey was stressed.

## Contract (what we promise)

| Rule | Behavior |
|------|----------|
| Quiesce | Returns iff every `Go`'d goroutine has **exited** or is parked in **Await** |
| Release | Requires a waiter at that transition (`ErrNoWaiter` otherwise) |
| Release | One-shot (`ErrAlreadyReleased` on double open) |
| Panic | Recovered inside `Go`; surfaced via `Panicked` / execute error |
| Explore/Replay | Fresh world per schedule; Release errors abort the candidate |

## Known limitations (not bugs to paper over)

| Situation | What happens |
|-----------|----------------|
| Blocked on `chan` recv/send (unbuffered, no peer) | **Not** quiescence → Quiesce hangs |
| Blocked on `sync.Mutex` | **Not** quiescence → Quiesce hangs |
| `context` cancel | Does **not** unblock `Await` |
| Worker skips a later gate (early return) | Next `Release` → `ErrNoWaiter` |
| Schedule names a gate never reached | `ErrNoWaiter` |
| Nested `Await` (sequential) | Supported |
| Channel traffic **between** released gates | Supported if handshake can finish before next Quiesce |

Quiesce is intentionally narrow: only `Await` counts as a controllable park.
That mirrors the synctest lesson — we do not pretend to observe the Go scheduler.

## Adversarial tests

```bash
go test ./internal/gate/ -run Adversarial -v
```
