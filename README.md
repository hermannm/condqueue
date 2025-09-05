# condqueue

A small Go package providing a concurrent queue, on which consumers can wait for an item satisfying
a given condition, and producers can add items to wake consumers.

Run `go get hermannm.dev/condqueue` to add it to your project!

**Docs:** [pkg.go.dev/hermannm.dev/condqueue](https://pkg.go.dev/hermannm.dev/condqueue)

**Contents:**

- [Usage](#usage)
- [Maintainer's guide](#maintainers-guide)

## Usage

In the example below, we have a simple message, which can either be of type success or error. A
producer goroutine adds messages to the queue, while two consumer goroutines wait for a message type
each.

<!-- @formatter:off -->
```go
import (
	"context"
	"fmt"
	"sync"

	"hermannm.dev/condqueue"
)

type Message struct {
	Type    string
	Content string
}

func main() {
	queue := condqueue.New[Message]()

	var wg sync.WaitGroup
	wg.Add(3)

	// Producer
	go func() {
		fmt.Println("[Producer] Adding success message...")
		queue.Add(Message{Type: "success", Content: "Great success!"})

		fmt.Println("[Producer] Adding error message...")
		queue.Add(Message{Type: "error", Content: "I've made a huge mistake"})

		wg.Done()
	}()

	// Consumer 1
	go func() {
		fmt.Println("[Consumer 1] Waiting for success...")

		msg, _ := queue.AwaitMatchingItem(context.Background(), func(candidate Message) bool {
			return candidate.Type == "success"
		})

		fmt.Printf("[Consumer 1] Received success message: %s\n", msg.Content)
		wg.Done()
	}()

	// Consumer 2
	go func() {
		fmt.Println("[Consumer 2] Waiting for errors...")

		msg, _ := queue.AwaitMatchingItem(context.Background(), func(candidate Message) bool {
			return candidate.Type == "error"
		})

		fmt.Printf("[Consumer 2] Received error message: %s\n", msg.Content)
		wg.Done()
	}()

	wg.Wait()
}
```
<!-- @formatter:on -->

This gives the following output (the order may vary due to concurrency):

```
[Consumer 2] Waiting for errors...
[Producer] Adding success message...
[Producer] Adding error message...
[Consumer 2] Received error message: I've made a huge mistake
[Consumer 1] Waiting for success...
[Consumer 1] Received success message: Great success!
```

For more details on how to use `condqueue`, refer to the
[documentation](https://pkg.go.dev/hermannm.dev/condqueue).

## Maintainer's guide

### Publishing a new release

- Run tests and linter ([`golangci-lint`](https://golangci-lint.run/)):
  ```
  go test ./... && golangci-lint run
  ```
- Add an entry to `CHANGELOG.md` (with the current date)
    - Remember to update the link section, and bump the version for the `[Unreleased]` link
- Create commit and tag for the release (update `TAG` variable in below command):
  ```
  TAG=vX.Y.Z && git commit -m "Release ${TAG}" && git tag -a "${TAG}" -m "Release ${TAG}" && git log --oneline -2
  ```
- Push the commit and tag:
  ```
  git push && git push --tags
  ```
    - Our release workflow will then create a GitHub release with the pushed tag's changelog entry
