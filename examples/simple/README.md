## Build

```bash
make
```

## Usage

Start in solo mode (single process)

```bash
./simple -mode solo
```

Start server

```bash
./simple -mode server
```

Start worker client (create as many as you want)

```bash
./simple -mode client
```

Configure number of workers in process

```bash
./simple -mode client -workers 10
```
