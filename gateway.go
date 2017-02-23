package deviot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type GatewayMode int

const (
	GATEWAY_MODE_HTTP_PULL GatewayMode = 0
	GATEWAY_MODE_HTTP_PUSH GatewayMode = 1
	GATEWAY_MODE_MQTT      GatewayMode = 2
)

type Gateway struct {
	Name          string             `json:"name"`
	Kind          string             `json:"kind"`
	Host          string             `json:"host"`
	Port          int                `json:"port"`
	Data          string             `json:"data"`
	Action        string             `json:"action"`
	Sensors       []Thing            `json:"sensors"`
	Mode          GatewayMode        `json:"mode"`
	Owner         string             `json:"owner"`
	Description   string             `json:"description,omitempty"`
	Things        map[string]wrapper `json:"-"`
	DevIoTServer  string             `json:"-"`
	mqttConnector MqttConnector      `json:"-"`
}

type wrapper struct {
	Instance Instance
	Thing    Thing
}

type Instance interface {
	GetThing() Thing
}

func NewGateway(name, kind, deviotServer string, host string, port int, opts map[string]interface{}) Gateway {
	mode := GATEWAY_MODE_MQTT
	if opts["mode"] != nil {
		mode = (opts["mode"]).(GatewayMode)
	}
	owner := ""
	if opts["owner"] != nil {
		owner = (opts["owner"]).(string)
	}
	description := ""
	if opts["description"] != nil {
		description = (opts["description"]).(string)
	}
	replacer := strings.NewReplacer("@", "-", ".", "_", "/", "_", ":", "_")
	data := fmt.Sprintf("/deviot/%s/%s/data/", replacer.Replace(owner), replacer.Replace(name))
	action := fmt.Sprintf("/deviot/%s/%s/action/", replacer.Replace(owner), replacer.Replace(name))
	gateway := Gateway{
		Name:         name,
		Kind:         kind,
		Host:         host,
		Port:         port,
		Data:         data,
		Action:       action,
		Sensors:      []Thing{},
		Mode:         mode,
		Owner:        owner,
		Description:  description,
		Things:       make(map[string]wrapper),
		DevIoTServer: deviotServer,
	}
	gateway.mqttConnector = NewMqttConnector(gateway, host, port, owner, action, data)
	return gateway
}

func (gateway Gateway) Start() error {
	err := gateway.mqttConnector.Start()
	if err != nil {
		return err
	}
	go func() {
		for {
			registerGateway(gateway.DevIoTServer, gateway)
			time.Sleep(5 * time.Second)
		}
	}()
	return err
}

func (gateway Gateway) Stop() error {
	return gateway.mqttConnector.Stop()
}

func (gateway Gateway) RegisterThing(instance Instance) {
	id := instance.GetThing().Id
	gateway.Things[id] = wrapper{Thing: instance.GetThing(), Instance: instance}
}

func (gateway Gateway) DeregisterThing(id string) {
	delete(gateway.Things, id)
}

func (gateway Gateway) SendData(data map[string]interface{}) error {
	return gateway.mqttConnector.Publish(data)
}

func (gateway Gateway) CallAction(data map[string]interface{}) {
	id, ok := data["id"]
	if !ok {
		id, ok = data["name"]
	}
	if !ok {
		fmt.Printf("Illegal message: id/name(%v) not found\n", id)
		return
	}
	wrapper, ok := gateway.Things[id.(string)]
	if !ok {
		fmt.Printf("Illegal message: id/name(%v) not found\n", id)
		return
	}
	action, ok := data["action"]
	if !ok {
		fmt.Printf("Illegal message: action(%v) not found\n", action)
		return
	}
	actionDef := wrapper.Thing.FindAction(action.(string))
	if !ok {
		fmt.Printf("Illegal message: action(%v) not found\n", action)
		return
	}
	method, ok := reflect.TypeOf(wrapper.Instance).MethodByName(action.(string))
	if !ok {
		fmt.Printf("Illegal message: action(%v) not found\n", action)
		return
	}
	args := make([]reflect.Value, len(actionDef.Parameters) + 1)
	args[0] = reflect.ValueOf(wrapper.Instance)
	for index, arg := range actionDef.Parameters {
		if value, ok := data[arg.Name]; ok {
			args[index + 1] = reflect.ValueOf(value)
		}
	}
	fmt.Printf("Calling action: %v %v\n", action, args)
	method.Func.Call(args)
}

func registerGateway(server string, gateway Gateway) {
	sensors := []Thing{}
	for _, v := range gateway.Things {
		sensors = append(sensors, v.Thing)
	}
	gateway.Sensors = sensors
	b, err := json.Marshal(&gateway)
	if err != nil {
		fmt.Printf("Fail to parse gateway to json: %s\n", err)
		return
	}
	registrationUrl := fmt.Sprintf("http://%s/api/v1/gateways", server)
	resp, error := http.Post(registrationUrl, "application/json", bytes.NewBuffer(b))
	if error != nil {
		fmt.Printf("Fail to register gateway. %s\n", error)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		fmt.Printf("Register gateway success(%d)\n", resp.StatusCode)
	} else {
		fmt.Printf("Fail to register gateway(%d)\n", resp.StatusCode)
	}
}
