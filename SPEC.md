# CacheStore contract

Language-neutral behaviour every implementation must provide. Names below are the Go spelling;
use the idiomatic casing elsewhere.

## Interface

```
Get(ctx, key)                    -> (value bytes, found bool, error)
Set(ctx, key, value, ttl)        -> error
SetNX(ctx, key, value, ttl)      -> (stored bool, error)
Delete(ctx, keys...)             -> error
Close()                          -> error
```

Values are opaque bytes. Typed (JSON) helpers `GetJSON`/`SetJSON` sit on top.

## Semantics

1. **Miss is not an error.** `Get` of an absent or expired key returns `found = false`, no error.
2. **An error means the store could not answer.** Callers treat it as a miss and carry on: a cache
   must never be the reason a request fails.
3. **TTL is mandatory and positive.** `Set` and `SetNX` reject `ttl <= 0` with `ErrInvalidTTL`. An
   entry that never expires is never what a cache wants.
4. **`SetNX` is the lease primitive.** It stores only when the key is absent or expired, and reports
   whether it did. The loser must not overwrite the holder's value.
5. **`Delete` of a missing key is not an error.**
6. **Values are copied** in and out of in-process stores; mutating a returned value must not change
   the stored one.
7. **`MemoryStore` is bounded** (default 10000 entries). When full it drops expired entries first,
   then the entry closest to expiry.
8. **`RedisStore` namespaces keys** under a caller-supplied prefix and does not close a client it
   was handed.
9. **`Fallback`**:
   - uses the primary while it works;
   - after a primary error, skips the primary for a retry window (default 5 s) and serves from the
     secondary, so requests do not each pay the primary's timeout;
   - on recovery the primary is the source of truth again; secondary entries are no longer
     consulted and expire on their own TTL;
   - `Delete` always clears the secondary too, so a deletion made during an outage cannot be
     resurrected by a stale local entry;
   - while degraded, `SetNX` leases are process-local;
   - exposes `Degraded()` and optional degraded/recovered hooks.
10. **`New` with no Redis client returns the memory store alone.**
11. **`GetJSON` treats an undecodable value as a miss**, not an error (the shape may have changed
    between deploys).

## Keys

Callers scope keys (tenant, organization, caller); the library does not know what they mean.
Never put a secret in a key: hash it (sha256 of a bearer token is the usual case). A shared store
makes an entry visible to every replica, which is the point for introspection results and the
hazard for anything caller-specific.

## Conformance cases

Every implementation's tests must cover, for each store where applicable: get/set/delete; expiry;
SetNX win, lose and win-after-expiry; invalid TTL; memory bound and value copying; Redis key
prefix; Fallback degrade, in-window skip, recovery, and delete clearing both sides; `New` with and
without a client; JSON round trip and undecodable-as-miss.
