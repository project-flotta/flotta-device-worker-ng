package job

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/robfig/cron"
	"github.com/tupyy/device-worker-ng/internal/entity"
)

type Builder struct {
	w            entity.Workload
	cronSpec     string
	retryBackoff backoff.BackOff
}

func NewBuilder(w entity.Workload) *Builder {
	return &Builder{w: w}
}

func (jb *Builder) WithExponentialRetry(initialInterval time.Duration, maxInterval time.Duration, multiplier float64) *Builder {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = initialInterval
	exp.MaxInterval = maxInterval
	exp.Multiplier = multiplier
	jb.retryBackoff = exp
	return jb
}

func (jb *Builder) WithConstantRetry(retryInterval time.Duration) *Builder {
	cst := backoff.NewConstantBackOff(retryInterval)
	jb.retryBackoff = cst
	return jb
}

func (jb *Builder) WithCron(cronSpec string) *Builder {
	jb.cronSpec = cronSpec
	return jb
}

func (jb *Builder) Build() (*DefaultJob, error) {
	j := &DefaultJob{
		workload:     jb.w,
		currentState: ReadyState,
		targetState:  ReadyState,
	}

	if jb.cronSpec != "" {
		specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		sched, err := specParser.Parse(jb.cronSpec)
		if err != nil {
			return nil, err
		}
		j.cron = &CronJob{
			schedule: sched,
			Next:     sched.Next(time.Now()),
		}
	}

	if jb.retryBackoff != nil {
		j.retry = &RetryJob{
			NextRetry: time.Now(),
			b:         jb.retryBackoff,
		}
	}

	return j, nil
}
