# adapter-contract

## ADDED Requirements

### Requirement: The engine sources adapters from a tool-id-keyed registry

The engine SHALL construct its set of tool adapters from a tool-id-keyed
registry of adapter factories, rather than a hardcoded list bound to each
adapter's concrete constructor. Registering a factory under a tool id MUST be
the only step required to add a built-in adapter; the engine MUST build every
registered adapter, in a deterministic order, passing each the shared
construction dependencies. Building from the registry MUST yield the same
adapters, with the same options, as the prior hardcoded wiring.

#### Scenario: Engine builds every registered adapter

- **WHEN** the engine builds its adapters
- **THEN** it builds one adapter per registered tool id, in registration order,
  each constructed from the shared dependencies — identical to the prior
  hardcoded set

#### Scenario: Adding an adapter is a registration

- **WHEN** a new adapter factory is registered under a new tool id
- **THEN** the engine includes it with no change to the engine's build logic
