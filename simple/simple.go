package simple

import (
	"fmt"
	"math/rand/v2"
	"time"
)

type Agent struct {
	id            int
	produces      string
	appleUtility  int
	bananaUtility int
	buyPrice      int
	sellPrice     int
	item          string
	coins         int
	state         string
}

func getItemName(index int) string {
	names := []string{"apple", "banana"}
	return names[index]
}
func otherItem(item string) string {
	if item == "apple" {
		return "banana"
	}
	return "apple"
}

var appleSellOffer chan int

var appleBuyOffer chan int
var bananaSellOffer chan int

var bananaBuyOffer chan int

func waitAndPublish(waitChannel chan int, waitTime int) {
	for range waitTime {
		time.Sleep(time.Second)
	}
	waitChannel <- 0
}
func (agent *Agent) attemptBuy(good string) bool {
	if agent.item != "none" {
		fmt.Printf("Agent %d inv not empty; buy failed\n", agent.id)
		return false
	}
	var sell int
	waitChannel := make(chan int)
	go waitAndPublish(waitChannel, 3)
	if good == "apple" {
		select {
		case sell = <-appleSellOffer:
			fmt.Printf("Agent %d fetched a %s sell offer for %d \n", agent.id, good, sell)
			appleBuyOffer <- agent.buyPrice
		case <-waitChannel:
			fmt.Printf("Agent %d buy took too long\n", agent.id)
			return false
		}

	} else {
		// good == "banana"
		select {
		case sell := <-bananaSellOffer:
			fmt.Printf("Agent %d fetched a %s sell offer for %d \n", agent.id, good, sell)
			bananaBuyOffer <- agent.buyPrice
		case <-waitChannel:
			fmt.Printf("Agent %d buy took too long\n", agent.id)
			return false
		}
	}
	agent.item = good
	exchange := (agent.buyPrice + sell) / 2
	agent.coins -= exchange
	fmt.Printf("Agent %d bought apple for %d\n", agent.id, exchange)
	return true
}

func (agent *Agent) attemptSell(good string) bool {
	if agent.item == "none" {
		fmt.Printf("Agent %d inv empty; sell failed\n", agent.id)
		return false
	}
	var buy int
	waitChannel := make(chan int)
	go waitAndPublish(waitChannel, 3)

	if good == "apple" {
		select {
		case appleSellOffer <- agent.sellPrice:
			fmt.Printf("Agent %d posted a %s sell offer for %d \n", agent.id, "apple", agent.sellPrice)
			buy = <-appleBuyOffer
		case <-waitChannel:
			fmt.Printf("Agent %d sell took too long\n", agent.id)
			return false
		}
	} else {
		// good == "banana"
		select {
		case bananaSellOffer <- agent.sellPrice:
			fmt.Printf("Agent %d posted a %s sell offer for %d \n", agent.id, "apple", agent.sellPrice)
			buy = <-bananaBuyOffer
		case <-waitChannel:
			fmt.Printf("Agent %d sell took too long\n", agent.id)
			return false
		}
	}

	exchange := (agent.sellPrice + buy) / 2
	agent.item = "none"
	agent.coins += exchange
	fmt.Printf("Agent %d sold %s at amount %d\n", agent.id, good, agent.sellPrice)
	return true
}

func runAgent(agent *Agent) {
	for {
		for range rand.IntN(1000) {
			time.Sleep(time.Millisecond)
		}
		if agent.item == "none" {
			fmt.Printf("Agent %d enqueuing to buy %s\n", agent.id, otherItem(agent.produces))
			success := agent.attemptBuy(otherItem(agent.produces))
			if !success {
				fmt.Printf("Agent %d producing %s\n", agent.id, agent.produces)
				agent.item = agent.produces
			}
			continue
		}

		if agent.item == agent.produces {
			fmt.Printf("Agent %d attempting to sell %s\n", agent.id, agent.produces)
			success := agent.attemptSell(agent.produces)
			if !success {
				fmt.Printf("Agent %d consuming its own %s\n", agent.id, agent.item)
				agent.item = "none"
			}
			continue
		}
		if (agent.item == "apple" && agent.produces == "banana") || (agent.item == "banana" && agent.produces == "apple") {
			fmt.Printf("Agent %d consuming %s\n", agent.id, agent.item)
			agent.item = "none"
		}
	}

}

func averagePrice(pool []*Agent, itemName string, action string) float32 {
	price := 0
	count := 0
	for _, a := range pool {
		if a.produces != itemName {
			continue
		}
		if action == "buy" {
			price += a.buyPrice
		} else {
			price += a.sellPrice
		}
		count += 1

	}
	return float32(price / count)
}

func Simulation() {

	bananaSellOffer = make(chan int)
	appleSellOffer = make(chan int)
	bananaBuyOffer = make(chan int)
	appleBuyOffer = make(chan int)
	var pool []*Agent = nil
	for i := range 5 {
		agent := Agent{
			id:            i,
			produces:      getItemName(i % 2),
			appleUtility:  5 - 3*(1-2*(i%2)), // if i is zero,then (1-2*(i%2)) is 1, and - 3 * 1 makes the first good worth 2
			bananaUtility: 5 + 3*(1-2*(i%2)), // if i is zero, then the second good is worth 8
			buyPrice:      rand.IntN(10),
			sellPrice:     rand.IntN(10),
			item:          "none",
			coins:         100,
			state:         "initialized",
		}

		fmt.Println(agent)
		pool = append(pool, &agent)
		go runAgent(&agent)
	}
	//averagePrice(pool, "apple", "buy")
	fmt.Println("---------")

	for {
		time.Sleep(time.Second * 2)
		numAppleProducersHoldingApple := 0
		numAppleProducersHoldingBanana := 0
		numBananaProducersHoldingApple := 0
		numBananaProducersHoldingBanana := 0
		for _, a := range pool {
			if a.produces == "apple" {
				if a.item == "apple" {
					numAppleProducersHoldingApple += 1
				} else if a.item == "banana" {
					numAppleProducersHoldingBanana += 1
				} // else no item

			} else {
				// a.produces == "banana"
				if a.item == "apple" {
					numBananaProducersHoldingApple += 1
				} else if a.item == "banana" {
					numBananaProducersHoldingBanana += 1
				} // else no item
			}
		}
		fmt.Println(numAppleProducersHoldingApple, numAppleProducersHoldingBanana, numBananaProducersHoldingApple, numBananaProducersHoldingBanana)
	}
}
