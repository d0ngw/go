package kafka

import (
	"testing"

	"github.com/Shopify/sarama"
	c "github.com/d0ngw/go/common"
	"github.com/stretchr/testify/assert"
)

func TestPartitioner(t *testing.T) {
	key := "aaa"
	var partitionNums int32 = 3
	p := NewKafkaDefaultPartitioner("")
	m := &sarama.ProducerMessage{Key: sarama.StringEncoder(key)}
	pn, err := p.Partition(m, partitionNums)
	assert.Equal(t, nil, err)
	assert.Equal(t, c.JString(key).HashCode()%partitionNums, pn)
}
