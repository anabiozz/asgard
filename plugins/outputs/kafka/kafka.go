package kafka

import (
	"crypto/tls"
	"fmt"
	"heimdall_project/asgard"
	"heimdall_project/asgard/plugins/outputs"
	"strings"
	"github.com/Shopify/sarama"
	"heimdall_project/asgard/plugins/serializers"
)

var ValidTopicSuffixMethods = []string{
	"",
	"measurement",
	"tags",
}

type (
	Kafka struct {
		// Kafka brokers to send metrics to
		Brokers []string
		// Kafka topic
		Topic string
		// Kafka topic suffix option
		TopicSuffix TopicSuffix `toml:"topic_suffix"`
		// Routing Key Tag
		RoutingTag string `toml:"routing_tag"`
		// Compression Codec Tag
		CompressionCodec int
		// RequiredAcks Tag
		RequiredAcks int
		// MaxRetry Tag
		MaxRetry int

		// Legacy SSL config options
		// TLS client certificate
		Certificate string
		// TLS client key
		Key string
		// TLS certificate authority
		CA string

		// Path to CA file
		SSLCA string `toml:"ssl_ca"`
		// Path to host cert file
		SSLCert string `toml:"ssl_cert"`
		// Path to cert key file
		SSLKey string `toml:"ssl_key"`

		// Skip SSL verification
		InsecureSkipVerify bool

		// SASL Username
		SASLUsername string `toml:"sasl_username"`
		// SASL Password
		SASLPassword string `toml:"sasl_password"`

		tlsConfig tls.Config
		producer  sarama.SyncProducer

		serializer serializers.Serializer
	}
	TopicSuffix struct {
		Method    string   `toml:"method"`
		Keys      []string `toml:"keys"`
		Separator string   `toml:"separator"`
	}
)

func (k *Kafka) GetTopicName(metric asgard.Metric) string {
	var topicName string
	switch k.TopicSuffix.Method {
	case "measurement":
		topicName = k.Topic + k.TopicSuffix.Separator + metric.Name()
	case "tags":
		var topicNameComponents []string
		topicNameComponents = append(topicNameComponents, k.Topic)
		for _, tag := range k.TopicSuffix.Keys {
			tagValue := metric.Tags()[tag]
			if tagValue != "" {
				topicNameComponents = append(topicNameComponents, tagValue)
			}
		}
		topicName = strings.Join(topicNameComponents, k.TopicSuffix.Separator)
	default:
		topicName = k.Topic
	}
	return topicName
}

func ValidateTopicSuffixMethod(method string) error {
	for _, validMethod := range ValidTopicSuffixMethods {
		if method == validMethod {
			return nil
		}
	}
	return fmt.Errorf("Unknown topic suffix method provided: %s", method)
}

func (k *Kafka) Connect() error {
	err := ValidateTopicSuffixMethod(k.TopicSuffix.Method)
	if err != nil {
		return err
	}
	config := sarama.NewConfig()

	config.Producer.RequiredAcks = sarama.RequiredAcks(k.RequiredAcks)
	config.Producer.Compression = sarama.CompressionCodec(k.CompressionCodec)
	config.Producer.Retry.Max = k.MaxRetry
	config.Producer.Return.Successes = true

	// Legacy support ssl config
	if k.Certificate != "" {
		k.SSLCert = k.Certificate
		k.SSLCA = k.CA
		k.SSLKey = k.Key
	}

	if err != nil {
		return err
	}

	if k.SASLUsername != "" && k.SASLPassword != "" {
		config.Net.SASL.User = k.SASLUsername
		config.Net.SASL.Password = k.SASLPassword
		config.Net.SASL.Enable = true
	}

	producer, err := sarama.NewSyncProducer(k.Brokers, config)
	if err != nil {
		return err
	}
	k.producer = producer
	return nil
}

func (k *Kafka) Close() error {
	return k.producer.Close()
}

func (k *Kafka) SetSerializer(serializer serializers.Serializer) {
	k.serializer = serializer
}

func (k *Kafka) Write(metrics []asgard.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	for _, metric := range metrics {
		fmt.Println(metric)
		buf, err := k.serializer.Serialize(metric)
		if err != nil {
			return err
		}

		topicName := k.GetTopicName(metric)

		m := &sarama.ProducerMessage{
			Topic: topicName,
			Value: sarama.ByteEncoder(buf),
		}
		if h, ok := metric.Tags()[k.RoutingTag]; ok {
			m.Key = sarama.StringEncoder(h)
		}

		_, _, err = k.producer.SendMessage(m)

		if err != nil {
			return fmt.Errorf("FAILED to send kafka message: %s\n", err)
		}
	}
	return nil
}

func init() {
	brokers := []string{"localhost:9092"}
	outputs.Add("kafka", func() asgard.Output {
		return &Kafka{
			Brokers:    brokers,
			Topic:      "test",
			MaxRetry:     3,
			RequiredAcks: -1,
		}
	})
}
