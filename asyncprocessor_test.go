package asyncprocessor

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	accumulatedVal := 0

	proc := NewProcess(
		WithBufferSize(100),
		WithTimeout(3*time.Second),
		WithCallback(func(v any) error {
			switch data := v.(type) {
			case int:
				accumulatedVal += data * 2
			case error:
				return data
			case nil:
				return errors.New("data is nil")
			default:
				return errors.New("incorrect data type")
			}
			return nil
		}),
	)

	for i := 1; i < 10; i++ {
		d := i
		proc.Queue(func() any {
			return doWork(d)
		})
	}
	proc.Launch()

	err := proc.Wait()
	if err != nil {
		fmt.Println(err.Error())
	}

	assert.Equal(t, 570, accumulatedVal)
	fmt.Printf("Accumulated value: %d\n", accumulatedVal)
}
