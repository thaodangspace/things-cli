package cli

import (
	"strings"
	"testing"
)

func TestBatchInputLimits(t *testing.T) {
	t.Run("line size", func(t *testing.T) {
		output, err := runBatchTest(t, &batchProbeService{}, strings.Repeat("x", batchMaxLineBytes+1)+"\n")
		if exitCodeFor(err) != exitUsage {
			t.Fatalf("exit=%d err=%v", exitCodeFor(err), err)
		}
		results := decodeBatchResults(t, output)
		if len(results) != 1 || results[0].Error == nil || results[0].Error.Code != "line_too_large" {
			t.Fatalf("results=%+v", results)
		}
	})

	t.Run("operation count", func(t *testing.T) {
		var input strings.Builder
		for i := 0; i < batchMaxOperations+1; i++ {
			input.WriteString(`{"operation":"complete","request":{"id":"task"}}`)
			input.WriteByte('\n')
		}
		output, err := runBatchTest(t, &batchProbeService{}, input.String())
		if exitCodeFor(err) != exitUsage {
			t.Fatalf("exit=%d err=%v", exitCodeFor(err), err)
		}
		results := decodeBatchResults(t, output)
		if len(results) != batchMaxOperations+1 || results[len(results)-1].Error == nil || results[len(results)-1].Error.Code != "too_many_operations" {
			t.Fatalf("result count=%d last=%+v", len(results), results[len(results)-1])
		}
	})
}
