package config

// Job is a public interface for job configuration.
type Job interface {
	// Name returns the job's name
	Name() string
	// Containers returns this job's containers
	Containers() []Container
}

type job struct {
	name   string
	Cntnrs []*container `json:"containers"`
}

func (t *job) Name() string {
	return t.name
}

func (t *job) Containers() []Container {
	containers := make([]Container, len(t.Cntnrs))
	for i, container := range t.Cntnrs {
		containers[i] = container
	}
	return containers
}
