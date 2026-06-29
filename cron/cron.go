package cron

import (
	"github.com/minhgiang16983/Minh-Kit-Hehe/metrics"
	gocron "github.com/go-co-op/gocron/v2"
)

type Cron struct {
	scheduler gocron.Scheduler
}

func NewCron(metrics *metrics.Metrics, opts ...gocron.SchedulerOption) (*Cron, error) {

	monitor := NewMonitorWithMetrics(metrics)
	if metrics != nil {
		opts = append(opts, gocron.WithMonitorStatus(monitor), gocron.WithMonitor(monitor))
	}
	scheduler, err := gocron.NewScheduler(opts...)
	if err != nil {
		return nil, err
	}
	scheduler.Start()
	return &Cron{
		scheduler: scheduler,
	}, nil
}

func (c *Cron) AddCronJob(name, cronExpression string, jobFn func(), jobParams ...any) error {

	cronJob := gocron.CronJob(cronExpression, true)
	task := gocron.NewTask(jobFn, jobParams...)
	_, err := c.scheduler.NewJob(cronJob, task, gocron.WithName(name))
	if err != nil {
		return err
	}
	return nil
}

func (c *Cron) Start() {
	c.scheduler.Start()
}

func (c *Cron) Shutdown() error {
	return c.scheduler.Shutdown()
}
