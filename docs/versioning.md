# Versioning

Flourish uses SemVer 2.0.0 for both runtime API versioning and git releases.

## Source of Truth

- version value is stored in `VERSION`.
- format is SemVer without leading `v` (example: `1.2.3`).

## Runtime API

Import the root module package:

```go
import "github.com/iw2rmb/flourish"
```

Available helpers:

- `flourish.Version()` -> returns `MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]`
- `flourish.VersionTag()` -> returns `v` + `flourish.Version()`
- `flourish.VersionIsSemver()` -> validates embedded version

## Git Release Flow

- release tags use `vMAJOR.MINOR.PATCH` (SemVer-compatible).
- tag value must match `VERSION`.
- GitHub workflow `.github/workflows/release.yml` enforces:
  - semver tag format
  - `VERSION`/tag consistency
  - full `go test ./...` pass before release

## Local Commands

Use `scripts/semver.sh`:

- `scripts/semver.sh show`
- `scripts/semver.sh set 1.2.3`
- `scripts/semver.sh bump patch`
- `scripts/semver.sh bump minor`
- `scripts/semver.sh bump major`
- `scripts/semver.sh tag`
- `scripts/semver.sh tag --push`

See also: `README.md`, `docs/README.md`.
