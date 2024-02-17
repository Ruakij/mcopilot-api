package browser

import "time"

func RunFuncNTimes(n int, delay time.Duration, f func() error) error {
	var err error
	for i := 0; i < n; i++ {
		err = f()
		if err == nil {
			// No error, break the loop
			break
		}
		time.Sleep(delay)
	}
	return err
}
