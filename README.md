# Simple distributed job queue

## Usage

Start in solo mode (single process)

```bash
./<binary> -mode solo
```

Start server

```bash
./<binary> -mode server
```

Start worker client (create as many as you want)

```bash
./<binary> -mode client
```

Configure number of workers in process

```bash
./<binary> -mode client -workers 10
```

## Example code

See [example](example/simple)

Create job struct and write decoder function

```golang
// CustomJob will only sleep for a configurable amount of time
type CustomJob struct {
  Delay time.Duration
}

// DecodeCustomJob is creates a instance of the job type from a byte array using gob
func DecodeCustomJob(content []byte) jobq.JobType {
  //your type here
  var jobIncoming CustomJob

  // start copying here
  b := bytes.Buffer{}
  b.Write(content)
  d := gob.NewDecoder(&b)
  err := d.Decode(&jobIncoming)
  if err != nil {
    log.Fatalf("failed decode job: %v", err)
  }

  // return decoded object
  return jobIncoming
}

// Process is the function that'll be called when a job is processed
func (j CustomJob) Process() {
  time.Sleep(j.Delay)
}
```

Register job type with decoder function

```golang
jobq.RegisterJob("MyCustomJob", DecodeCustomJob)
```

Parse Arguments and start Job queue

```golang
jobq.Flags()
flag.Parse()
mode := jobq.Start()
defer jobq.Stop()
```

Enqueue job

```golang
job := CustomJob{1 * time.Second}
jobq.EnqueueJob("MyCustomJob", job)
```
