package influx

import "github.com/anabiozz/asgard"

type InfluxSerializer struct {}

func (s *InfluxSerializer) Serialize(m asgard.Metric) ([]byte, error) {
	return m.Serialize(), nil
}
