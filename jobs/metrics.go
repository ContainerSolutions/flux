package jobs

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/ContainerSolutions/flux"
	fluxmetrics "github.com/ContainerSolutions/flux/metrics"
)

type instrumentedJobStore struct {
	js JobStore
}

var (
	requestDuration = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Namespace: "flux",
		Subsystem: "jobs",
		Name:      "request_duration_seconds",
		Help:      "Request duration in seconds.",
		Buckets:   stdprometheus.DefBuckets,
	}, []string{fluxmetrics.LabelMethod, fluxmetrics.LabelSuccess})
	jobDuration = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Namespace: "flux",
		Subsystem: "jobs",
		Name:      "job_duration_seconds",
		Help:      "Job duration in seconds.",
		Buckets:   stdprometheus.DefBuckets,
	}, []string{fluxmetrics.LabelMethod, fluxmetrics.LabelSuccess})
)

func InstrumentedJobStore(js JobStore) JobStore {
	return &instrumentedJobStore{
		js: js,
	}
}

func (i *instrumentedJobStore) GetJob(inst flux.InstanceID, jobID JobID) (j Job, err error) {
	defer func(begin time.Time) {
		requestDuration.With(
			fluxmetrics.LabelMethod, "GetJob",
			fluxmetrics.LabelSuccess, fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return i.js.GetJob(inst, jobID)
}

func (i *instrumentedJobStore) PutJob(inst flux.InstanceID, j Job) (jobID JobID, err error) {
	defer func(begin time.Time) {
		requestDuration.With(
			fluxmetrics.LabelMethod, "PutJob",
			fluxmetrics.LabelSuccess, fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return i.js.PutJob(inst, j)
}

func (i *instrumentedJobStore) PutJobIgnoringDuplicates(inst flux.InstanceID, j Job) (jobID JobID, err error) {
	defer func(begin time.Time) {
		requestDuration.With(
			fluxmetrics.LabelMethod, "PutJobIgnoringDuplicates",
			fluxmetrics.LabelSuccess, fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return i.js.PutJobIgnoringDuplicates(inst, j)
}

func (i *instrumentedJobStore) UpdateJob(j Job) (err error) {
	defer func(begin time.Time) {
		requestDuration.With(
			fluxmetrics.LabelMethod, "UpdateJob",
			fluxmetrics.LabelSuccess, fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return i.js.UpdateJob(j)
}

func (i *instrumentedJobStore) Heartbeat(jobID JobID) (err error) {
	defer func(begin time.Time) {
		requestDuration.With(
			fluxmetrics.LabelMethod, "Heartbeat",
			fluxmetrics.LabelSuccess, fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return i.js.Heartbeat(jobID)
}

func (i *instrumentedJobStore) NextJob(queues []string) (j Job, err error) {
	defer func(begin time.Time) {
		requestDuration.With(
			fluxmetrics.LabelMethod, "NextJob",
			fluxmetrics.LabelSuccess, fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return i.js.NextJob(queues)
}

func (i *instrumentedJobStore) GC() (err error) {
	defer func(begin time.Time) {
		requestDuration.With(
			fluxmetrics.LabelMethod, "GC",
			fluxmetrics.LabelSuccess, fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return i.js.GC()
}
