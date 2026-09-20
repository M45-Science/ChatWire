# ChatWire scheduled-reboot failure evidence — 2026-09-20 UTC

Collected read-only at approximately 16:46 UTC. No service was restarted or signaled.

## Affected instances

| instance | PID | started | state / wait | threads | shutdown log ended |
|---|---:|---|---|---:|---|
| cw-c | 249801 | 2026-09-13 10:01:59 | sleeping / `futex_do_wait` | 26 | 10:01:47 |
| cw-e | 246827 | 2026-09-13 10:00:40 | sleeping / `futex_do_wait` | 26 | 10:00:27 |
| cw-h | 247514 | 2026-09-13 10:01:09 | sleeping / `futex_do_wait` | 27 | 10:00:57 |
| cw-i | 247695 | 2026-09-13 10:01:19 | sleeping / `futex_do_wait` | 27 | 10:01:08 |
| cw-k | 248007 | 2026-09-13 10:01:30 | sleeping / `futex_do_wait` | 29 | 10:01:17 |
| cw-m | 249617 | 2026-09-13 10:01:49 | sleeping / `futex_do_wait` | 25 | 10:01:37 |
| cw-p | 251533 | 2026-09-13 10:02:59 | sleeping / `futex_do_wait` | 25 | 10:02:27 |
| cw-r | 251224 | 2026-09-13 10:02:49 | sleeping / `futex_do_wait` | 25 | 10:02:17 |

All eight systemd units report `ActiveState=active`, `SubState=running`, `Result=success`, and still retain the old PID. All use `Restart=always` with `RestartUSec=1s`. Their cgroups are populated and not frozen.

## Common shutdown sequence

Each affected ChatWire log records the same successful sequence:

1. SIGUSR1 restart request accepted.
2. Factorio `/quit` sent.
3. Map saved.
4. Factorio stdout closed and process exit observed.
5. `restart-chatwire` follow-up executed.
6. Autolaunch disabled.
7. Database save and log close reported.
8. stdout/journal prints `Goodbye.`
9. ChatWire process remains alive indefinitely; systemd therefore never performs its configured restart.

The Factorio child processes for these instances are gone. The stuck process is ChatWire itself.

## Live process evidence

- Every inspected thread in every affected process reports kernel wait channel `futex_do_wait`.
- No pending process signals were reported.
- Each process retains its journald stdout/stderr socket, cgroup `cpu.max` descriptor, Go runtime epoll/eventfd descriptors, the new daily audit log, and one TCP socket.
- The daily audit log retained after normal log closure is 74 bytes and contains only the file-open banner; the primary ChatWire log stops at `Closing log files.`
- Resident memory is approximately 35–43 MiB per process at collection time.

## Strong code-path indication

`fact.DoExit` performs the following final operations in order:

```go
fmt.Println("Goodbye.")
if disc.DS != nil {
    disc.DS.Close()
}
os.Exit(1)
```

Because `Goodbye.` is present for all eight processes, but none reached `os.Exit(1)`, the blocking call is strongly localized to `disc.DS.Close()` (or code synchronously invoked by it). The retained TCP socket and futex-sleeping threads are consistent with a Discord session close/wait deadlock.

## Evidence locations

- Per-instance ChatWire logs: `/home/fact2/cw-{c,e,h,i,k,m,p,r}/audit-log/cw-20-September-2026.log`
- Per-instance post-close audit logs: `/home/fact2/cw-{c,e,h,i,k,m,p,r}/audit-log/audit-20-September-2026.log`
- Shutdown implementation: `/home/fact2/github/ChatWire/fact/util.go` (`DoExit`)
- Units: `/etc/systemd/system/chatwire-{c,e,h,i,k,m,p,r}.service`

## Collection limitations

The current account cannot read the full system journal or `/proc/*/syscall`, and `gdb`/`dlv` is not installed. No SIGQUIT goroutine dump was taken because Go's default SIGQUIT handling would terminate the affected process and destroy the live failure state.
