# tools/

[Up](../README.md)

Utility packages for the Upsilon ecosystem.

## Packages

| Package | File | Description |
|---|---|---|
| `tools` | `tools.go` | Math utilities: ranges, distances, interpolation |
| `actor` | `actor/` | Actor model for single-goroutine resource ownership |
| `messagequeue` | `messagequeue/` | Serialized message queue used by the actor |
| `messagequeue/message` | `messagequeue/message/` | Message envelope type |

## Dependency Graph

```
actor
 └── messagequeue
      └── message
```

The `tools` root package has no internal dependencies.
