package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

var (
	action    = flag.String("action", "toggle", "One of (toggle, open, test, party, vacation)")
	id        = flag.Int64("id", 123456, "Transmitter ID")
	button_id = flag.Int64("button_id", 2, "(2:primary, 1:secondary, 0:none)")
	version   = flag.Int64("version", 1, "(1: remote, 2:keypad, 9:sensor)")
	count     = flag.Int64("count", 10, "Number of times to transmit the code")
)

func decodeAction(action string) (option, command int64, err error) {
	switch action {
	case "toggle":
		command = 3
	case "open":
		command = 1
	case "party":
		option = 8
	case "vacation":
		option = 9
	case "test":
		option = 15
	default:
		return 0, 0, fmt.Errorf("Unknown action: %q", action)
	}
	return
}

func codeFromFlags(actionName string) (int64, error) {
	option, command, err := decodeAction(actionName)
	if err != nil {
		return 0, err
	}
	// TODO : range checks
	var code int64
	code |= *version << 38
	code |= option << 34
	code |= command << 30
	code |= *button_id << 22
	code |= *id
	return code, nil
}

func toBits(code int64) (out string) {
	v := code
	for i := 0; i < 42; i++ {
		if v%2 == 0 {
			out = "1000" + out
		} else {
			out = "1110" + out
		}
		v /= 2
	}
	return out
}

func main() {
	c := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	// hostname, _ := os.Hostname()

	mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New(os.Stdout, "", 0)
	opts := mqtt.NewClientOptions()
	opts.AddBroker("mqtt://homeassistant.local:1883")
	opts.SetClientID("sullyhausrf")
	opts.SetUsername("mqtt_user")
	opts.SetPassword("mqtt_password")
	opts.SetDefaultPublishHandler(f)
	opts.SetAutoReconnect(true)
	opts.SetCleanSession(false)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	fmt.Printf("Connected to %s\n", "homeassistant.local")

	subscribeCallback := func(c mqtt.Client, m mqtt.Message) {
		msg := Message{}
		err := json.Unmarshal(m.Payload(), &msg)
		if err != nil {
			fmt.Println("Failed to parse message: " + string(m.Payload()))
			return
		}

		if len(msg.Action) > 0 {
			fmt.Println("Action pressed: " + msg.Action)
			code, actionErr := codeFromFlags(msg.Action)
			if actionErr != nil {
				fmt.Printf("ERROR: Invalid action provided `%s`\n", msg.Action)
			}

			bits := toBits(code)

			fmt.Printf("Code: {42}%011x\n", code*4)
			fmt.Printf("Binary: %b\n", code)
			fmt.Printf("After pwm: %q\n", bits)

			cmd := exec.Command("sudo", "sendook", "-1", "250", "-0", "250", "-r", strconv.FormatInt(*count, 10), "-p", "40000", bits)
			cmdErr := cmd.Run()
			if cmdErr != nil {
				fmt.Println(cmdErr)
			}
		}
	}

	// main listener for "gate/action" mqtt requests
	if token := client.Subscribe("sullyhausrf/gate/action", 0, subscribeCallback); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	<-done
}
