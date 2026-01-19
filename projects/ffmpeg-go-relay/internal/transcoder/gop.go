package transcoder

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func gopArgs(gop string) ([]string, error) {
	gop = strings.TrimSpace(gop)
	if gop == "" {
		return nil, nil
	}

	if frames, err := strconv.Atoi(gop); err == nil {
		if frames <= 0 {
			return nil, fmt.Errorf("gop must be a positive frame count or duration")
		}
		return []string{"-g", strconv.Itoa(frames)}, nil
	}

	if dur, err := time.ParseDuration(gop); err == nil {
		if dur <= 0 {
			return nil, fmt.Errorf("gop must be a positive frame count or duration")
		}
		seconds := dur.Seconds()
		return []string{"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%.3f)", seconds)}, nil
	}

	return nil, fmt.Errorf("gop must be a positive frame count or duration")
}
