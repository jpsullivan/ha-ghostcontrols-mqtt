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

func toBits(code int64) string {
	var buf [168]byte
	v := code
	for i := 41; i >= 0; i-- {
		pat := "1000"
		if v&1 == 1 {
			pat = "1110"
		}
		copy(buf[i*4:i*4+4], pat)
		v >>= 1
	}
	return string(buf[:])
}

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)

	mqtt.ERROR = log.New(os.Stderr, "", 0)
	opts := mqtt.NewClientOptions()
	opts.AddBroker("mqtt://homeassistant.local:1883")
	opts.SetClientID("sullyhausrf")
	opts.SetUsername("mqtt_user")
	opts.SetPassword("mqtt_password")
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
				return
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

	<-sig
	client.Disconnect(250)
}
