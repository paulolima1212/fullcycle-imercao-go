package main

import (
	"encoding/json"
	"sync"

	"github.com/fullcycle/imercao/go/internal/infra/kafka"
	"github.com/fullcycle/imercao/go/internal/market/dto"
	"github.com/fullcycle/imercao/go/internal/market/entity"
	"github.com/fullcycle/imercao/go/internal/market/transformer"

	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
	orderIn := make(chan *entity.Order)
	orderOut := make(chan *entity.Order)
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	kafkaMsgChan := make(chan *ckafka.Message)

	configMap := &ckafka.ConfigMap{
		"bootstrap.servers": "host.docker.internal:9094",
		"group.id":          "myGroup",
		"auto.offset.reset": "earliest",
	}

	producer := kafka.NewKafkaProducer(configMap)

	kafka := kafka.NewConsumer(configMap, []string{"input"})

	go kafka.Consume(kafkaMsgChan)

	book := entity.NewBook(orderIn, orderOut, wg)

	go book.Trade()

	go func() {
		for msg := range kafkaMsgChan {
			wg.Add(1)
			tradeInput := dto.TradeInput{}
			err := json.Unmarshal(msg.Value, &tradeInput)
			if err != nil {
				panic(err)
			}
			order := transformer.TransformerInput(tradeInput)
			orderIn <- order
		}
	}()

	for res := range orderOut {
		output := transformer.TransformerOutput(res)
		outputJson, err := json.Marshal(output)

		if err != nil {
			panic(err)
		}

		producer.Publish(outputJson, []byte("orders"), "output")
	}
}
