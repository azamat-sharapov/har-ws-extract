This CLI tool extracts WebSocket messages from given HAR file and generates request-response mappings in JSON format.

# Usage

```
go run main.go -input filename.har -url wss://example.com -matcher-key id > output.json
```

`output.json`:

```
{
	"01e0dc92099df6b2fc2a224cdbf9267fa6d08584": "{{\"result\":\"ok\"}",
	"252eda843443456c9ae7aa043333580ae484ef0d": "true",
}
```

# How does it work?

It searches for WebSocket messages from source (`url`) in given HAR file (`input`) and matches each WebSocket `send message` (request) with `receive message` (response) by matcher (`matcher-key`). Then it serializes "request" into compact format and creates hash of it using SHA1 algorithm. That hash is used as output JSON key. Generated output will be `{"[sha1 encrypted request]": "[stringified response]"}`.

This is useful for mocking WebSocket messages on client. Generated JSON can be used as "indexed storage of WebSocket messages".

For consistent output, it sorts JSON object keys when serializing, otherwise hashes would be different on each run. Below is JavaScript implementation of serialization:

```javascript
function serialize (obj) {
	if (Array.isArray(obj)) {
		return obj
			.map(i => serialize(i))
			.join(",")
	}
	else if (typeof obj === 'object' && obj !== null) {
		return Object.keys(obj)
			.sort()
			.map(k => {
				const v = obj[k]
				if (v instanceof Object && v.constructor === Object) {
					return `${k}.${serialize(v)}`
				}
				else {
					return `${k}:${serialize(v)}`
				}
			})
			.join("|")
	}

	return obj
}
```
