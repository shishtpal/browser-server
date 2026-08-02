package tools

import (
	"bytes"
	"fmt"
)

// rawPythonResult formats a pythonResult as a readable raw-output block.
// It returns (nil, false) when v is not a pythonResult so the registry falls
// back to JSON marshaling safely.
func rawPythonResult(v any) ([]byte, bool) {
	res, ok := v.(pythonResult)
	if !ok {
		return nil, false
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "# stderr: %q\n", res.Stderr)
	fmt.Fprintf(&b, "# exit_code: %d\n", res.ExitCode)
	fmt.Fprintf(&b, "# timed_out: %t\n", res.TimedOut)
	fmt.Fprintf(&b, "# duration_ms: %d\n", res.DurationMs)
	if res.StdoutTruncated {
		fmt.Fprintln(&b, "# stdout_truncated: true")
	}
	if res.StderrTruncated {
		fmt.Fprintln(&b, "# stderr_truncated: true")
	}
	b.WriteByte('\n')
	b.WriteString(res.Stdout)
	return b.Bytes(), true
}
