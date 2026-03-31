package dot

import "fmt"

func ExampleMq() {
	client := NewAmqpClient("amqp://guest:guest@10.64.115.96:5672/")
	ch := client.Channel()
	//err := ch.Publish("", "dy-device", false, false, amqp.Publishing{ContentType: "text/plain", Body:[]byte("123")})
	//fmt.Println(err)

	msg, ok, err := ch.Get("dy-device", true)
	fmt.Println(string(msg.Body), ok, err)

	// Output: xx
}
