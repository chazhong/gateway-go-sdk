package deviot

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"time"
)

type MqttConnector struct {
	client mqtt.Client
	Action string
	Data   string
}

func NewMqttConnector(gateway Gateway, host string, port int, clientId string, action string, data string) MqttConnector {
	mqttUrl := fmt.Sprintf("tcp://%s:%d", host, port)
	opts := mqtt.NewClientOptions().AddBroker(mqttUrl).SetClientID(clientId)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		if msg.Topic() == action {
			fmt.Printf("Message received: %s\n", msg.Payload())
			var data map[string]interface{}
			if err := json.Unmarshal(msg.Payload(), &data); err != nil {
				fmt.Printf("Invalid message: %s\n", msg.Payload())
			} else {
				gateway.CallAction(data)
			}
		}
	})
	client := mqtt.NewClient(opts)
	connector := MqttConnector{client: client, Action: action, Data: data}
	return connector
}

func (connector MqttConnector) Start() error {
	if token := connector.client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to connect to MQ. Error: %s\n", token.Error())
		return token.Error()
	} else {
		fmt.Printf("MQ is connected\n")
		connector.subscribe(connector.Action)
		return nil
	}
}

func (connector MqttConnector) Stop() error {
	connector.client.Disconnect(250)
	return nil
}

func (connector MqttConnector) subscribe(queue string) error {
	if token := connector.client.Subscribe(queue, 0, nil); token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to subscribe to %s. Error: %s\n", queue, token.Error())
		return token.Error()
	} else {
		fmt.Printf("%s is subscribed\n", queue)
		return nil
	}
}

func (connector MqttConnector) Publish(data map[string]interface{}) error {
	bytes, _ := json.Marshal(data)
	text := string(bytes)
	if token := connector.client.Publish(connector.Data, 0, false, text); token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to publish %s to %s. Error: %s\n", text, connector.Data, token.Error())
		return token.Error()
	} else {
		fmt.Printf("Message %s is published\n", text)
		return nil
	}
}
