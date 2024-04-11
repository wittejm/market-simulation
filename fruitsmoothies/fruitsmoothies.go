package fruitsmoothies

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
)

type Agent struct {
	id                int
	strategy          [16]float32
	item              string
	buyFruitPrice     int
	buySmoothiePrice  int
	sellFruitPrice    int
	sellSmoothiePrice int
	hunger            int
	coins             int
}

func printAgentSummary(pool []*Agent) {
	hunger := 0
	coinsSum := 0
	var buyFruitPrices []int
	var sellFruitPrices []int
	var allCoins []int
	for _, a := range pool {
		hunger += a.hunger
		coinsSum += a.coins
		allCoins = append(allCoins, a.coins)
		buyFruitPrices = append(buyFruitPrices, a.buyFruitPrice)
		sellFruitPrices = append(sellFruitPrices, a.sellFruitPrice)

	}
	slices.Sort(buyFruitPrices)
	//fmt.Println("buys:", buyFruitPrices)
	slices.Sort(sellFruitPrices)
	slices.Sort(allCoins)
	fmt.Println("coins:", allCoins)

	fmt.Println(hunger, coinsSum)

}

var fruitSellOffer chan int
var fruitBuyOffer chan int
var smoothieSellOffer chan int
var smoothieBuyOffer chan int

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
	var buy int
	waitChannel := make(chan int)

	go waitAndPublish(waitChannel, 1)
	if good == "fruit" {

		select {
		case sell = <-fruitSellOffer:
			//fmt.Printf("Agent %d fetched a %s sell offer for %d \n", agent.id, good, sell)
			buy = agent.buyFruitPrice
			fruitBuyOffer <- buy
		case <-waitChannel:
			//fmt.Printf("Agent %d buy took too long\n", agent.id)
			return false
		}
	} else {
		// good == "smoothie"
		select {
		case sell = <-smoothieSellOffer:
			//fmt.Printf("Agent %d fetched a %s sell offer for %d \n", agent.id, good, sell)
			buy = agent.buySmoothiePrice
			smoothieBuyOffer <- buy
		case <-waitChannel:
			//fmt.Printf("Agent %d buy took too long\n", agent.id)
			return false
		}
	}
	if sell <= buy {
		agent.item = good
		exchange := (buy + sell) / 2
		agent.coins -= exchange
		fmt.Printf("Agent %d bought %s for %d\n", agent.id, good, exchange)
		if good == "fruit" {
			agent.buyFruitPrice -= 1
		} else {
			agent.buySmoothiePrice -= 1
		}
		return true
	} else {
		//fmt.Printf("Agent %d buy failed, sale price too high\n", agent.id)
		if good == "fruit" {
			agent.buyFruitPrice += 1
		} else {
			agent.buySmoothiePrice += 1
		}
		return false
	}
}

func (agent *Agent) attemptSell(good string) bool {
	if agent.item == "none" {
		fmt.Printf("Agent %d inv empty; sell failed\n", agent.id)
		return false
	}
	var buy int
	var sell int
	waitChannel := make(chan int)
	go waitAndPublish(waitChannel, 1)

	if good == "fruit" {
		sell = agent.sellFruitPrice
		select {
		case fruitSellOffer <- sell:
			//fmt.Printf("Agent %d posted a %s sell offer for %d \n", agent.id, good, sell)
			buy = <-fruitBuyOffer
		case <-waitChannel:
			//fmt.Printf("Agent %d sell took too long\n", agent.id)
			return false
		}
	} else {
		sell = agent.sellSmoothiePrice
		select {
		case smoothieSellOffer <- sell:
			// fmt.Printf("Agent %d posted a %s sell offer for %d \n", agent.id, good, sell)
			buy = <-smoothieBuyOffer
		case <-waitChannel:
			// fmt.Printf("Agent %d sell took too long\n", agent.id)
			return false
		}
	}

	if sell <= buy {
		if good == "fruit" {
			agent.sellFruitPrice += 1
		} else {
			agent.sellSmoothiePrice += 1
		}
		exchange := (sell + buy) / 2
		agent.item = "none"
		agent.coins += exchange
		fmt.Printf("Agent %d sold %s for %d\n", agent.id, good, exchange)
		return true

	} else {
		//fmt.Printf("Agent %d sale failed, buy price too low\n", agent.id)
		if good == "fruit" {
			agent.sellFruitPrice -= 1
		} else {
			agent.sellSmoothiePrice -= 1
		}
		return false
	}

}

func (agent *Agent) selectAction() {
	// Agent strategy has these states:
	// If no item and hunger == none
	// If no item and hunger == low
	// If no item and hunger == medium
	// If no item and hunger == high
	// (for each, 3 actions: grow fruit, buy fruit, buy smoothie)
	// If holding fruit and hunger == none
	// If holding fruit and hunger == low
	// If holding fruit and hunger == medium
	// If holding fruit and hunger == high
	// (for each, 2 actions: make smoothie, sell fruit)
	// If holding smoothie and hunger == none
	// If holding smoothie and hunger == low
	// If holding smoothie and hunger == medium
	// If holding smoothie and hunger == high
	// (for each, 2 actions: consume smoothie, sell smoothie)
	// That's 16 variables. Three options is represented by 2 variables. Two options has a single variable.
	var hungerIndex int
	if agent.hunger < 2500 {
		hungerIndex = 0
	} else if agent.hunger < 5000 {
		hungerIndex = 1
	} else if agent.hunger < 7500 {
		hungerIndex = 2
	} else {
		hungerIndex = 3
	}

	if agent.item == "none" {
		if rand.Float32() < agent.strategy[hungerIndex] {
			agent.growFruit()
		} else if rand.Float32() < agent.strategy[4+hungerIndex] {
			agent.attemptBuy("fruit")
		} else {
			agent.attemptBuy("smoothie")
		}
	} else if agent.item == "fruit" {
		if rand.Float32() < agent.strategy[8+hungerIndex] {
			agent.makeSmoothie()
		} else {
			agent.attemptSell("fruit")
		}
	} else {
		// has a smoothie
		if rand.Float32() < agent.strategy[12+hungerIndex] {
			agent.consumeSmoothie()
		} else {
			agent.attemptSell("smoothie")
		}
	}
}

func (agent *Agent) growFruit() {
	//fmt.Printf("Agent %d growing fruit\n", agent.id)
	time.Sleep(1 * time.Second) // Todo, make an agent have "grow time" specialization
	agent.item = "fruit"
}

func (agent *Agent) makeSmoothie() {
	//fmt.Printf("Agent %d making smoothie\n", agent.id)
	time.Sleep(1 * time.Second) // TODO ^^
	agent.item = "smoothie"
}

func (agent *Agent) consumeSmoothie() {
	//fmt.Printf("Agent %d consuming smoothie\n", agent.id)
	time.Sleep(1 * time.Second) // TODO ^^
	agent.item = "none"
	agent.hunger = max(agent.hunger-10000, 0)
}

func (agent *Agent) run() {
	for {
		for range rand.IntN(100) {
			time.Sleep(time.Millisecond)
		}
		agent.selectAction()
	}
}

func Simulation() {

	fmt.Println("starting simulation")

	fruitSellOffer = make(chan int)
	fruitBuyOffer = make(chan int)
	smoothieSellOffer = make(chan int)
	smoothieBuyOffer = make(chan int)

	var pool []*Agent = nil
	for i := range 10 {
		strategy := [16]float32{}
		for j := range 16 {
			strategy[j] = rand.Float32()
		}
		newAgent := Agent{
			id:                i,
			strategy:          strategy,
			item:              "none",
			buyFruitPrice:     rand.IntN(10),
			buySmoothiePrice:  rand.IntN(10),
			sellFruitPrice:    rand.IntN(10),
			sellSmoothiePrice: rand.IntN(10),
			hunger:            0,
			coins:             100,
		}

		pool = append(pool, &newAgent)
		go (&newAgent).run()
	}
	for {
		time.Sleep(time.Second)
		printAgentSummary(pool)
		for _, a := range pool {
			a.hunger += 100
		}
	}
}
