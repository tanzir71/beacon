# U1 Native Default-Scheduling Gate Stability

Date: 2026-11-25. Status: open acceptance limitation, not whole-goal completion.

Three default-scheduled full-gate validation attempts failed intermittently on
Windows, despite independent focused tests and one traced package run passing.
Do not accept a cached run or an unchanged retry as proof this is resolved.

## Evidence

1. The first failure had no completed folder-burst snapshot and a command that
   did not reach its second run. The then-pending UI entrypoint was moved out
   of the checkout; the baseline was clean at `13d8a57` before diagnosis.
2. The single traced native server suite passed in 22.292s, with zero shared-TEMP
   watches. This excluded neither scheduling delays nor slow publication.
3. After notification hardening, the unchanged three-second burst check failed
   with all 20 apps and one rebuild already published. Diagnostics showed one
   reconciliation still active, no native/reconciliation error, and no pending
   debounce. The test did not observe the existing 600ms stable window before
   its deadline; late publication versus delayed observation is not separated.
4. After cancellation-order fixes, the unchanged restart check failed with a
   starting page and `ManagerBusy:true`. Notification work was not running and
   its last duration was 8.9867ms. The manager was busy closing/starting/checking
   the replacement; exact child/OS timing is not yet established.

Operation diagnostics now report that work and bounded logs without waiting for
the manager lock. Watcher reports retain event/debounce/reconciliation facts and
errors. Neither test's assertions, timeout, or intra-package parallelism changed.

## Serialized Package Comparison

The full native `make check` passed with process-local `GOFLAGS=-p=1`, using
serialized Go package execution. This retains full lint, race instrumentation,
every test and its internal parallelism/deadline, version injection, all zero-CGO
targets, tray build and shipped-file checks. No source gate or CI flags changed.

That passing comparison did not prove that package scheduling caused the earlier
failures. A later run with the same GOFLAGS also failed SSE, rename, debounce
and slow-command timing checks (server package 194.548s). Serialization alone
is not a reliable workaround. Do not present it as unqualified acceptance or
mark U1/B0 complete because of it.

## Isolated Linux Validation

The restored UI service passed the unchanged full gate in a local Linux Docker
container with Go 1.27.4, Node 24, Python 3.11, PHP 8.2 and golangci-lint 2.15.0.
The container had four CPUs, 6 GiB memory, an executable temporary filesystem,
no network access, no published ports and no writable host-source mount. The
source was copied in; Git executable modes were restored inside that copy.
No Go test scheduling flags, assertions or deadlines were changed for this run.

Initial container setup was corrected rather than changing tests: Debian needed
the `python` alias, `--init` was needed to reap orphaned command fixtures, and
Windows-to-container copy had changed a tracked HTML file from 644 to 755,
causing the existing executable detector to classify it as a command. Restoring
the Git index modes fixed that reproducible transfer artifact. The corrected
gate passed, including server race tests in 10.280s and all three zero-CGO builds.

This is valid Linux evidence and a practical local development path. It is not
Windows acceptance and does not establish the cause of native host timing drift.
The native default-scheduling criterion remains open.

## Independently Verified Hardening

- A deterministic test failed before repair because a blocked change callback
  blocked the restart caller. Notifications now use one owned coalescing worker.
- Shutdown cancels work before waiting on startup locks/watchers, joins startup,
  monitor and notification workers, and preserves callback failures for owners.
- Failed constructors roll back acquired resources. These behaviors have direct
  race-detector proofs and are not asserted to explain both intermittent tests.

## Exit Evidence Still Needed

Capture an uncached default-scheduled failure with the new phase/log diagnostics
and fix a demonstrated cause, then verify default scheduling without altering
deadlines or assertions. A reliable native execution environment can also supply
acceptance evidence. GitHub Actions remain prohibited by the owner's instruction.
Independent U1 work may continue with the disclosed local orchestration result;
this acceptance limitation must stay open until evidence closes it.
