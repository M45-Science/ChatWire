# ChatWire–SoftMod protocol

ChatWire and SoftMod use one system-only Factorio command and one output tag:

```text
/chatwire <base64(deflate(json))>
[CHATWIRE] <base64(deflate(json))>
```

The codec matches Factorio's `helpers.encode_string` and
`helpers.decode_string`. JSON is retained as the inner format so envelopes can
be versioned and inspected after decoding without maintaining a custom codec.
This is transport encoding, not encryption; it keeps routine console output
compact and less casually readable but does not provide secrecy.

Requests contain `v`, `id`, `command`, and `data`. Responses echo `id` and
`command` and contain `ok`, `data`, and an optional `error`. Unsolicited output
uses `kind: "event"`, an `event` name, and `data`.

Protocol version 1 supports `hello`, `status`, `config`, `online`, `chat`,
`whisper`, `player-level`, and `supporter`. The command rejects in-game players;
it is intended only for the server console or RCON. Player and moderator-facing
SoftMod commands, including the in-game `/online` window, are separate and are
not removed by this protocol.
