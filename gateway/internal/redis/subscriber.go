package redis

import (
	"encoding/json"
	"log"
)

func StartSubscriber(handler func(Event)) {
	go func() {
		pubsub := Client.Subscribe(Ctx, "relay-events")
		defer pubsub.Close()

		ch := pubsub.Channel()
		log.Println("Subscribed to relay-events")

		for msg := range ch {
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Println("Invalid event payload:", err)
				continue
			}
			handler(event)
		}
	}()
}
