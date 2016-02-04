// 提供一些kafka的工具类
package kafka

import (
	"github.com/Shopify/sarama"
	c "github.com/d0ngw/go/common"
)

// kafka_abs 是Kafka自己实现的求绝对值的方式
func kafkaAbs(n int32) int32 {
	return n & 0x7fffffff
}

type kafkaDefaultPartitioner struct {
	saramPartitioner sarama.Partitioner
}

func (self *kafkaDefaultPartitioner) RequiresConsistency() bool {
	return true
}

// NewKafkaDefaultPartitioner 创建一个部分兼容kafka.producer.DefaultPartitioner的分区算法
//
// 1) key是string,int32和int64,使用kafka.producer.DefaultPartitioner的算法,即使用java.lang.Object.hashcode做hash取摸
//
// 2) key是其他类型,使用sarama.NewHashPartitioner的算法
func NewKafkaDefaultPartitioner(topic string) sarama.Partitioner {
	p := sarama.NewHashPartitioner(topic)
	return &kafkaDefaultPartitioner{p}
}

func (self *kafkaDefaultPartitioner) Partition(message *sarama.ProducerMessage, numPartitions int32) (int32, error) {
	key := message.Key
	if key == nil {
		return self.saramPartitioner.Partition(message, numPartitions)
	}
	var jt c.JTypeCompatible = nil
	switch v := key.(type) {
	case sarama.StringEncoder:
		jt = c.JString(v)
	default:
		jt = nil
	}
	if jt != nil {
		hashCode := jt.HashCode()
		return kafkaAbs(hashCode) % numPartitions, nil
	} else {
		return self.saramPartitioner.Partition(message, numPartitions)
	}
}
