# Changelog

### v0.5.1 (2025-12-16)

* Added example and documentation for using actors to protect struct state,
  making methods thread-safe.

### v0.5.0 (2025-12-16)

* Added comprehensive examples for documentation and tests
* Added benchmarks for key functions
* Added fuzz tests for actor actions
* Added a concurrency test to ensure race-free execution
* Updated `doc.go` and `README.md` for better readability and more examples
* Updated `Makefile` with a `fuzz` target

### v0.4.0 (2025-05-23)

* Migrated to Go 1.24
* Improved error handling for Actions
* Added typed result values

### v0.3.0 (2023-04-08)

* Migrated to Go 1.19
* Added Repeat() methods for running background Actions in intervals
* Added context to individual Action calls
* Removed Action calls with timeouts
* Changed to common queue for synchronous and asynchronous Actions
* Improved handling of timeouts and cancellations via contexts
* Improved external checking if Actor is still running

### v0.2.0 (2022-05-18)

* Migrated to Go 1.18

### v0.1.0 (2021-09-01)

* Migrated from Tideland Together
