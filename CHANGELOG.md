# Changelog

### v1.0.0 (2025-12-22)

* BREAKING: Version 1.0.0 marks the stabilization of the generic actor API
* Added `DoSync()`, `DoSyncWithError()`, and `DoSyncWithErrorContext()`
  methods for synchronous queueing with optional error handling
* Added `DoAsync()`, `DoAsyncWithError()`, and `DoAsyncWithErrorContext()`
  methods for asynchronous queueing with optional error handling
* Added `DoAsyncAwait()`, `DoAsyncAwaitWithError()`, and `DoAsyncAwaitWithErrorContext()`
  methods for async queueing with deferred synchronous waiting
* Added `Query()` and `Update()` methods for convenient state access
* Changed API to encapsulate Actor state inside the Actor and access
  only via its methods
* Added example and documentation for using actors to protect struct state,
  making methods thread-safe.
* Added comprehensive examples for documentation and tests
* Added benchmarks for key functions
* Added fuzz tests for actor actions
* Added a concurrency test to ensure race-free execution
* Updated `doc.go` and `README.md`, and `HOWTO.md` for better readability and more examples
* Addded a `Makefile`

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
