# xk6-nats

This is a [k6](https://go.k6.io/k6) extension using the [xk6](https://github.com/k6io/xk6) system, that allows to use NATS protocol.

|  ❗ This extension isn't supported by the k6 team, and may break in the future. USE AT YOUR OWN RISK! |
|------|

- [xk6-nats](#xk6-nats)
  - [Build](#build)
  - [API](#api)
    - [Nats](#nats)
    - [Publishing](#publishing)
    - [Subscribing](#subscribing)
  - [License](#license)

## Build

To build a `k6` binary with this extension, first ensure you have the prerequisites:

- [Go toolchain](https://go101.org/article/go-toolchain.html)
- Git

1. Install `xk6` framework for extending `k6`:
```shell
go install go.k6.io/xk6/cmd/xk6@latest
```

2. Build the binary:
```shell
xk6 build --with github.com/ydarias/xk6-nats@latest
```

3. Run a test

```shell
k6 run -e NATS_HOSTNAME=localhost test/test.js
```

To run JetStream test, make sure NATS JetStream is started, e.g. `nats-server -js`

```shell
k6 run -e NATS_HOSTNAME=localhost test/test_jetstream.js
```

To run publish with headers test, make sure NATS JetStream is started, e.g. `nats-server -js`

```shell
./k6 run -e NATS_HOSTNAME=localhost test/test_headers.js
```

## API

## Nats

A `Nats` instance represents the connection with the NATS server. It's created with `new Nats(configuration)`, where `configuration` is an object with the following attributes:

| Attribute | Description |
| --- | --- |
| `servers` | (mandatory) A list of NATS server URLs (e.g., `['nats://localhost:4222']`). |
| `auth` | (optional) An object defining the authentication strategy. |

### `auth` Configuration

The `auth` object allows you to specify how to authenticate with the NATS server. It has the following attributes:

| Attribute | Description |
| --- | --- |
| `strategy` | (optional) The authentication strategy to use. Can be `'unsafe'`, `'user_password'`, or `'token'`. If not provided, no specific authentication is used, but `unsafe` (for TLS) can still apply. |
| `unsafe` | (optional) A boolean value (default: `false`). If `true`, it allows connecting to NATS with self-signed TLS certificates, useful for testing environments. This applies when `strategy` is `'unsafe'` or no `strategy` is specified. |
| `token` | (optional) The authentication token used to connect to the NATS server. Required if `strategy` is `'token'`. |
| `username` | (optional) The username for `'user_password'` authentication. Required if `strategy` is `'user_password'`. |
| `password` | (optional) The password for `'user_password'` authentication. Required if `strategy` is `'user_password'`. |

**Example Usage for Connection:**

```typescript
import { Nats } from 'k6/x/nats';

// Example: Basic connection with unsafe TLS
const natsConfigUnsafe = {
    servers: ['nats://localhost:4222'],
    auth: {
        unsafe: true,
        strategy: 'unsafe', // Explicitly declare 'unsafe' strategy
    },
};
const natsUnsafe = new Nats(natsConfigUnsafe);

// Example: Connection with username and password
const natsConfigUserPass = {
    servers: ['nats://localhost:4222'],
    auth: {
        strategy: 'user_password',
        username: 'myuser',
        password: 'mypassword',
    },
};
const natsUserPass = new Nats(natsConfigUserPass);

// Example: Connection with a token
const natsConfigToken = {
    servers: ['nats://localhost:4222'],
    auth: {
        strategy: 'token',
        token: 'mysecrettoken',
    },
};
const natsToken = new Nats(natsConfigToken);
```

---
### Publishing

You can publish messages to a topic using the following functions:

| Function | Description |
| --- | --- |
| `publish(topic, payload)` | publish a new message using the topic (string) and the given payload that is a string representation that later is serialized as a byte array |
| `publisWithHeaders(topic, payload, headers)` | publish a new message using the topic (string), the given payload that is a string representation that later is serialized as a byte array and the headers |
| `publishMsg(message)` | publish a new message using the `message` (object) that has the following attributes: `topic` (string), `data` (string), `raw`(byte array) and `headers` (object) |
| `request(topic, payload, headers)` | sends a request to the topic (string) and the given payload as string representation and the headers, and returns a `message` |

Example:

```ts
const publisher = new Nats(natsConfig)

publisher.publish('topic', 'data')
publisher.publishWithHeaders('topic', 'data', { 'header1': 'value1' })
publisher.publishMsg({ topic: 'topic', data: 'string data', headers: { 'header1': 'value1' } })
publisher.publishMsg({ topic: 'topic', raw: [ 0, 1, 2, 3 ], headers: { 'header1': 'value1' } })
const message = publisher.request('topic', 'data', { 'header1': 'value1' })
```

### Subscribing

You can subscribe to a topic using the following functions:

| Function | Description |
| --- | --- |
| `subscribe(topic, callback)` | subscribe to a topic (string) and execute the callback function when a `message` is received, it returns a `subscription` |

Example:

```ts
const subscriber = new Nats(natsConfig)
const subscription = subscriber.subscribe('topic', (msg) => {
    console.log(msg.data)
})
// ...
subscription.close()
```

> **Note:** the subscription model has been changed. Now when you use `subscribe` method, a subscription object is returned and the subscription should be closed using the `close()` method.

## License

The source code of this project is released under the [MIT License](LICENSE).