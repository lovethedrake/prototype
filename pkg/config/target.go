package config

// Target is a public interface for target configuration.
type Target interface {
	// Name returns the target's name
	Name() string
	// SidecarContainers returns this target's containers
	Containers() []Container
}

type target struct {
	name   string
	Cntnrs []*container `json:"containers"`
}

func (t *target) Name() string {
	return t.name
}

func (t *target) Containers() []Container {
	containers := make([]Container, len(t.Cntnrs))
	for i, container := range t.Cntnrs {
		containers[i] = container
	}
	return containers
}
