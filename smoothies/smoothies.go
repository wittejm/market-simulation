package smoothies

/*
Traders attempt direct agent to agent sale with short wait time. Lots of
failing trade attempts.
*/
import (
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
)

// Agent something
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
	fmt.Println("buys:", buyFruitPrices)
	slices.Sort(sellFruitPrices)
	fmt.Println("sells:", sellFruitPrices)
	slices.Sort(allCoins)
	fmt.Println("coins:", allCoins)

	fmt.Println(hunger, coinsSum)

}

var fruitSellOffer chan int
var fruitBuyOffer chan int
var smoothieSellOffer chan int
var smoothieBuyOffer chan int

var fruitBuyerAgent chan *Agent
var fruitSellerAgent chan *Agent
var smoothieBuyerAgent chan *Agent
var smoothieSellerAgent chan *Agent

var agentChannels map[*Agent](chan int)

func fruitExchangeLoop() {
	fruitBuyerAgent = make(chan *Agent)
	fruitSellerAgent = make(chan *Agent)

	for {
		var buyer *Agent
		var seller *Agent
		expired := time.After(500 * time.Millisecond)
		select {
		case buyer = <-fruitBuyerAgent:
			fmt.Printf("got a buyer for fruit: %d\n", buyer.id)
			select {
			case seller = <-fruitSellerAgent:
				fmt.Printf("got a seller for fruit: %d\n", seller.id)
			case <-expired:
				continue
			}
		case <-expired:
			continue
		}
		price := (buyer.buyFruitPrice + seller.sellFruitPrice) / 2
		if buyer.coins < price {
			continue
		} else if buyer.buyFruitPrice < seller.sellFruitPrice {
			fmt.Println("Fruit sale failed because price mismatch")
			buyer.buyFruitPrice++
			seller.sellFruitPrice--
			continue
		} else {
			buyer.item = seller.item
			seller.item = "none"
			buyer.coins -= price
			seller.coins += price
			buyer.buyFruitPrice--
			seller.sellFruitPrice++
			fmt.Printf("Completed a fruit sale between %d %d\n", buyer.id, seller.id)

		}
		agentChannels[buyer] <- 0
		agentChannels[seller] <- 0
	}
}

func smoothieExchangeLoop() {
	smoothieBuyerAgent = make(chan *Agent)
	smoothieSellerAgent = make(chan *Agent)

	for {
		var buyer *Agent
		var seller *Agent
		expired := time.After(500 * time.Millisecond)
		select {
		case buyer = <-smoothieBuyerAgent:
			fmt.Printf("got a buyer for smoothie: %d\n", buyer.id)
			select {
			case seller = <-smoothieSellerAgent:
				fmt.Printf("got a seller for smoothie: %d\n", seller.id)
			case <-expired:
				continue
			}
		case <-expired:
			continue
		}
		price := (buyer.buySmoothiePrice + seller.sellSmoothiePrice) / 2
		if buyer.coins < price {
			continue
		} else if buyer.buySmoothiePrice < seller.sellSmoothiePrice {
			fmt.Println("Smoothie sale failed because price mismatch")
			buyer.buySmoothiePrice++
			seller.sellSmoothiePrice--
		} else {
			buyer.item = seller.item
			seller.item = "none"
			buyer.coins -= price
			seller.coins += price
			fmt.Printf("Completed a smoothie sale between %d %d\n", buyer.id, seller.id)
			buyer.buySmoothiePrice--
			seller.sellSmoothiePrice++

		}
		agentChannels[buyer] <- 0
		agentChannels[seller] <- 0
	}
}

func (agent *Agent) attemptBuy(good string) bool {
	if agent.item != "none" {
		fmt.Printf("Agent %d inv not empty; buy failed\n", agent.id)
		return false
	}

	expired := time.After(500 * time.Millisecond)

	if good == "fruit" {
		select {
		case fruitBuyerAgent <- agent:
			fmt.Printf("Agent %d posted a buy attempt %s \n", agent.id, good)
			select {
			case <-agentChannels[agent]:
				fmt.Println("exchange return signal received")
				return true
			case <-expired:
				return false
			}
		case <-expired:
			return false
		}
	} else {
		// good = "smoothie"
		select {
		case smoothieBuyerAgent <- agent:
			fmt.Printf("Agent %d posted a buy attempt %s \n", agent.id, good)
			select {
			case <-agentChannels[agent]:
				fmt.Println("exchange return signal received")
				return true
			case <-expired:
				//fmt.Printf("Agent %d buy took too long\n", agent.id)
				return false
			}
		case <-expired:
			fmt.Printf("Agent %d buy for %s took too long\n", agent.id, good)
			return false
		}
	}
}

func (agent *Agent) attemptSell(good string) bool {
	if agent.item == "none" {
		fmt.Printf("Agent %d inv empty; sell failed\n", agent.id)
		return false
	}

	expired := time.After(500 * time.Millisecond)
	if good == "fruit" {
		select {
		case fruitSellerAgent <- agent:
			fmt.Printf("Agent %d posted to sell %s\n", agent.id, good)
			select {
			case <-agentChannels[agent]:
				fmt.Println("exchange return signal received")
				return true
			case <-expired:
				fmt.Printf("Agent %d sell for fruit took too long\n", agent.id)
				return false
			}
		case <-expired:
			fmt.Printf("Agent %d sell for fruit took too long\n", agent.id)
			return false
		}
	} else {

		select {
		case fruitBuyerAgent <- agent:
			fmt.Printf("Agent %d posted to sell %s\n", agent.id, good)
			select {
			case <-agentChannels[agent]:
				fmt.Println("exchange return signal received")
				return true
			case <-expired:
				fmt.Printf("Agent %d buy took too long\n", agent.id)
				return false
			}
		case <-expired:
			fmt.Printf("Agent %d buy took too long\n", agent.id)
			return false
		}
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
		if rand.Float32()+0.3 < agent.strategy[hungerIndex] {
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
		if rand.Float32()-0.3 < agent.strategy[12+hungerIndex] {
			agent.consumeSmoothie()
		} else {
			agent.attemptSell("smoothie")
		}
	}
}

func (agent *Agent) growFruit() {
	fmt.Printf("Agent %d growing fruit\n", agent.id)
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
	time.Sleep(1 * time.Second)
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

// Simulation here is a description
func Simulation() {

	fmt.Println("starting simulation")

	go fruitExchangeLoop()
	go smoothieExchangeLoop()

	var pool []*Agent = nil
	agentChannels = make(map[*Agent](chan int))
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
		agentChannels[&newAgent] = make(chan int)
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
