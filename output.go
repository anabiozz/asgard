package asgard

type Output interface {
	// Connect to the Output
	Connect() error
	// Close any connections to the Output
	Close() error
	// Write takes in group of points to be written to the Output
	Write(metrics []Metric) error
}

type ServiceOutput interface {
	// Connect to the Output
	Connect() error
	// Close any connections to the Output
	Close() error
	// Write takes in group of points to be written to the Output
	Write(metrics []Metric) error
	// Start the "service" that will provide an Output
	Start() error
	// Stop the "service" that will provide an Output
	Stop()
}