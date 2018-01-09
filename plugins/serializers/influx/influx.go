package influx

import "heimdall_project/asgard"

type InfluxSerializer struct {}

func (s *InfluxSerializer) Serialize(m asgard.Metric) ([]byte, error) {
	return m.Serialize(), nil
}
