# config-model

## ADDED Requirements

### Requirement: Config loading runs as explicit ordered phases

Config loading SHALL run as explicit, ordered phases — decode (parse + schema
version guard), migrate (legacy-form folding), normalize (defaulting), and
validate — rather than as a single monolithic function. Each phase MUST run in
that order, and the observable result (the loaded config, and every validation
error) MUST be identical to the prior monolithic loader.

#### Scenario: Loading a config runs decode, migrate, normalize, validate in order

- **WHEN** a config is loaded
- **THEN** it is decoded, migrated, normalized, and validated in that order, and
  the resulting config and any error are identical to the prior loader
