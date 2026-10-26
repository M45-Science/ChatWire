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

The first creation of a map's SoftMod storage marks a preview pending, including saves where Factorio ran `on_init` before SoftMod injection. On the first successful `hello`, SoftMod emits `map-preview` with `data.exchange_string` from `game.get_map_exchange_string()`, then clears the persistent pending flag. Existing maps, configuration changes and repeated handshakes do not announce again. ChatWire posts a FactMapGen share link to that server's chat channel. The link contains the actual compressed exchange data, including the resolved seed; it does not look up a preset. Unusually long links are attached as `map-preview.txt` rather than truncated.

`/map-preview` is also available on demand in Factorio and Discord. Factorio replies privately to the requesting player, or to RCON when called from the console. Discord invokes this fixed read-only command over RCON and returns an ephemeral link (or a text attachment for exceptionally long links). Register the new Discord slash command during the planned rollout; it requires the updated SoftMod to be loaded. Manual requests do not consume or repeat the automatic new-map announcement.
