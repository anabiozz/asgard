package models

import (
	"heimdall_project/asgard"
	"heimdall_project/asgard/metric"
	"log"
	"math"
	"strings"
	"time"
)

// makemetric is used by both RunningAggregator & RunningInput
// to make metrics.
//   nameOverride: override the name of the measurement being made.
//   namePrefix:   add this prefix to each measurement name.
//   nameSuffix:   add this suffix to each measurement name.
//   pluginTags:   these are tags that are specific to this plugin.
//   daemonTags:   these are daemon-wide global tags, and get applied after pluginTags.
//   filter:       this is a filter to apply to each metric being made.
//   applyFilter:  if false, the above filter is not applied to each metric.
//                 This is used by Aggregators, because aggregators use filters
//                 on incoming metrics instead of on created metrics.
// TODO refactor this to not have such a huge func signature.
func makemetric(measurement string, fields map[string]interface{}, tags map[string]string,
	mType asgard.ValueType, t time.Time) asgard.Metric {

	if len(fields) == 0 || len(measurement) == 0 {
		return nil
	}
	if tags == nil {
		tags = make(map[string]string)
	}

	for k, v := range tags {
		if strings.HasSuffix(k, `\`) {
			log.Printf("D! Measurement [%s] tag [%s] "+
				"ends with a backslash, skipping", measurement, k)
			delete(tags, k)
			continue
		} else if strings.HasSuffix(v, `\`) {
			log.Printf("D! Measurement [%s] tag [%s] has a value "+
				"ending with a backslash, skipping", measurement, k)
			delete(tags, k)
			continue
		}
	}

	for k, v := range fields {
		if strings.HasSuffix(k, `\`) {
			log.Printf("D! Measurement [%s] field [%s] "+
				"ends with a backslash, skipping", measurement, k)
			delete(fields, k)
			continue
		}
		// Validate uint64 and float64 fields
		// convert all int & uint types to int64
		switch val := v.(type) {
		case nil:
			// delete nil fields
			delete(fields, k)
		case uint:
			fields[k] = int64(val)
			continue
		case uint8:
			fields[k] = int64(val)
			continue
		case uint16:
			fields[k] = int64(val)
			continue
		case uint32:
			fields[k] = int64(val)
			continue
		case int:
			fields[k] = int64(val)
			continue
		case int8:
			fields[k] = int64(val)
			continue
		case int16:
			fields[k] = int64(val)
			continue
		case int32:
			fields[k] = int64(val)
			continue
		case uint64:
			// InfluxDB does not support writing uint64
			if val < uint64(9223372036854775808) {
				fields[k] = int64(val)
			} else {
				fields[k] = int64(9223372036854775807)
			}
			continue
		case float32:
			fields[k] = float64(val)
			continue
		case float64:
			// NaNs are invalid values in influxdb, skip measurement
			if math.IsNaN(val) || math.IsInf(val, 0) {
				log.Printf("D! Measurement [%s] field [%s] has a NaN or Inf "+
					"field, skipping",
					measurement, k)
				delete(fields, k)
				continue
			}
		case string:
			fields[k] = v
		default:
			fields[k] = v
		}
	}

	m, err := metric.New(measurement, tags, fields, t, mType)
	if err != nil {
		log.Printf("Error adding point [%s]: %s\n", measurement, err.Error())
		return nil
	}

	return m
}
