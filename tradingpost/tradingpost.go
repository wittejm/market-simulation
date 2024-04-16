package tradingpost

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"sort"
	"time"
)

// Agent comment
type Agent struct {
	id                   int
	strategy             [16]float32
	item                 string
	buyFruitPrice        int
	buySmoothiePrice     int
	sellFruitPrice       int
	sellSmoothiePrice    int
	hunger               int
	coins                int
	actionCompleted      chan int
	numFruitBuys         int
	numFruitSales        int
	numSmoothieBuys      int
	numSmoothieSales     int
	numFruitGrowth       int
	numSmoothiesMade     int
	numSmoothiesConsumed int
}

// The TradingPost comment
type TradingPost struct {
	coins             int
	fruit             int
	smoothies         int
	buyFruitPrice     int
	sellFruitPrice    int
	buySmoothiePrice  int
	sellSmoothiePrice int
	buyFruitQueue     chan *Agent
	sellFruitQueue    chan *Agent
	buySmoothieQueue  chan *Agent
	sellSmoothieQueue chan *Agent
}

var tradingPost TradingPost
var simulationSpeed int // a multiplier

func runTradingPost() {
	tradingPost.coins = 1000000
	tradingPost.fruit = 100
	tradingPost.smoothies = 100
	tradingPost.buyFruitPrice = rand.IntN(9)
	tradingPost.sellFruitPrice = tradingPost.buyFruitPrice + 1
	tradingPost.buySmoothiePrice = rand.IntN(9)
	tradingPost.sellSmoothiePrice = tradingPost.buyFruitPrice + 1
	tradingPost.buyFruitQueue = make(chan *Agent)
	tradingPost.sellFruitQueue = make(chan *Agent)
	tradingPost.buySmoothieQueue = make(chan *Agent)
	tradingPost.sellSmoothieQueue = make(chan *Agent)

	// Selling fruit to buyers
	go func() {
		for {
			var nextFruitBuyer *Agent = <-tradingPost.buyFruitQueue
			failure := checkTradeConditions(nextFruitBuyer, "fruit", "buy")
			if failure {
				nextFruitBuyer.actionCompleted <- 1
				continue
			}
			exchange := (nextFruitBuyer.buyFruitPrice + tradingPost.sellFruitPrice) / 2
			tradingPost.coins += exchange
			nextFruitBuyer.coins -= exchange
			tradingPost.fruit--
			nextFruitBuyer.item = "fruit"
			nextFruitBuyer.numFruitBuys++
			logAgentAction(nextFruitBuyer.id, fmt.Sprintf("Agent %d completed fruit purchase for %d coins\n", nextFruitBuyer.id, exchange))
			updateTradePostPrices("sell", "fruit", exchange)
			nextFruitBuyer.updatePrices("buy", "fruit", exchange)
			if tradingPost.fruit < 50 {
				updateTradePostPrices("sell", "fruit", 1000) // Increase the sale price
			}
			if tradingPost.fruit > 150 {
				updateTradePostPrices("sell", "fruit", 0) // Reduce the sale price
			}
			if tradingPost.fruit > 250 {
				updateTradePostPrices("sell", "fruit", 0) // Reduce the sale price
			}
			if tradingPost.fruit > 350 {
				updateTradePostPrices("sell", "fruit", 0) // Reduce the sale price
			}

			nextFruitBuyer.actionCompleted <- 1
		}
	}()
	// Buying fruit from sellers
	go func() {
		for {
			var nextFruitSeller *Agent = <-tradingPost.sellFruitQueue
			failure := checkTradeConditions(nextFruitSeller, "fruit", "sell")
			if failure {
				nextFruitSeller.actionCompleted <- 1
				continue
			}
			exchange := (nextFruitSeller.sellFruitPrice + tradingPost.buyFruitPrice) / 2
			tradingPost.coins -= exchange
			nextFruitSeller.coins += exchange

			tradingPost.fruit++
			nextFruitSeller.item = "none"
			nextFruitSeller.numFruitSales++

			logAgentAction(nextFruitSeller.id, fmt.Sprintf("Agent %d completed fruit sale for %d coins\n", nextFruitSeller.id, exchange))
			updateTradePostPrices("buy", "fruit", exchange)
			nextFruitSeller.updatePrices("sell", "fruit", exchange)
			if tradingPost.fruit < 50 {
				updateTradePostPrices("buy", "fruit", 1000) // Increase the buy price
			}
			if tradingPost.fruit > 150 {
				updateTradePostPrices("buy", "fruit", 0) // Reduce the buy price
			}
			if tradingPost.fruit > 250 {
				updateTradePostPrices("buy", "fruit", 0) // Reduce the buy price
			}
			if tradingPost.fruit > 350 {
				updateTradePostPrices("buy", "fruit", 0) // Reduce the buy price
			}

			nextFruitSeller.actionCompleted <- 1
		}
	}()
	// Selling smoothies to buyers
	go func() {
		for {
			var nextSmoothieBuyer *Agent = <-tradingPost.buySmoothieQueue
			failure := checkTradeConditions(nextSmoothieBuyer, "smoothie", "buy")
			if failure {
				nextSmoothieBuyer.actionCompleted <- 1
				continue
			}
			exchange := (nextSmoothieBuyer.buySmoothiePrice + tradingPost.sellSmoothiePrice) / 2
			tradingPost.coins += exchange
			nextSmoothieBuyer.coins -= exchange
			tradingPost.smoothies--
			nextSmoothieBuyer.item = "smoothie"
			nextSmoothieBuyer.numSmoothieBuys++
			logAgentAction(nextSmoothieBuyer.id, fmt.Sprintf("Agent %d completed smoothie purchase for %d coins\n", nextSmoothieBuyer.id, exchange))
			updateTradePostPrices("sell", "smoothie", exchange)
			nextSmoothieBuyer.updatePrices("buy", "smoothie", exchange)
			if tradingPost.smoothies < 50 {
				updateTradePostPrices("sell", "smoothie", 1000) // Increase the sale price
			}
			if tradingPost.smoothies < 50 {
				updateTradePostPrices("sell", "smoothie", 1000) // Increase the buy price
			}
			if tradingPost.smoothies > 150 {
				updateTradePostPrices("sell", "smoothie", 0) // Reduce the buy price
			}
			nextSmoothieBuyer.actionCompleted <- 1
		}
	}()
	// Buying smoothies from sellers
	go func() {
		for {
			var nextSmoothieSeller *Agent = <-tradingPost.sellSmoothieQueue
			failure := checkTradeConditions(nextSmoothieSeller, "smoothie", "sell")
			if failure {
				nextSmoothieSeller.actionCompleted <- 1
				continue
			}
			exchange := (nextSmoothieSeller.sellSmoothiePrice + tradingPost.buySmoothiePrice) / 2
			tradingPost.coins -= exchange
			nextSmoothieSeller.coins += exchange
			tradingPost.smoothies++
			nextSmoothieSeller.item = "none"
			nextSmoothieSeller.numSmoothieSales++
			logAgentAction(nextSmoothieSeller.id, fmt.Sprintf("Agent %d completed smoothie sale for %d coins\n", nextSmoothieSeller.id, exchange))
			updateTradePostPrices("buy", "smoothie", exchange)
			nextSmoothieSeller.updatePrices("sell", "smoothie", exchange)
			if tradingPost.smoothies < 50 {
				updateTradePostPrices("buy", "smoothie", 1000) // Increase the buy price
			}
			if tradingPost.smoothies > 150 {
				updateTradePostPrices("buy", "smoothie", 0) // Reduce the buy price
			}
			nextSmoothieSeller.actionCompleted <- 1
		}
	}()
}

func updateTradePostPrices(action, good string, exchange int) {

	if action == "buy" {
		if good == "fruit" {
			if tradingPost.buyFruitPrice < (exchange) {
				tradingPost.buyFruitPrice++
				logTradeFailure(-1, "post buy fruit price increase\n")

			}
			if tradingPost.buyFruitPrice > (exchange) {
				logTradeFailure(-1, "post buy fruit price decrease\n")
				tradingPost.buyFruitPrice--
			}
		}
		if good == "smoothie" {
			if tradingPost.buySmoothiePrice < (exchange) {
				tradingPost.buySmoothiePrice++
				logTradeFailure(-1, "post buy smoothie price increase\n")

			}
			if tradingPost.buySmoothiePrice > (exchange) {
				tradingPost.buySmoothiePrice--
				logTradeFailure(-1, "post buy smoothie price decrease\n")
			}
		}
	}
	if action == "sell" {
		if good == "fruit" {
			if tradingPost.sellFruitPrice < exchange {
				tradingPost.sellFruitPrice++
				logTradeFailure(-1, "post sell fruit price increase\n")
			}
			if tradingPost.sellFruitPrice > exchange {
				tradingPost.sellFruitPrice--
				logTradeFailure(-1, "post sell fruit price decrease\n")

			}
		}
		if good == "smoothie" {
			if tradingPost.sellSmoothiePrice < exchange {
				tradingPost.sellSmoothiePrice++
				logTradeFailure(-1, "post sell smoothie price increase\n")

			}
			if tradingPost.sellSmoothiePrice > exchange {
				tradingPost.sellSmoothiePrice--
				logTradeFailure(-1, "post sell smoothie price decrease\n")
			}
		}
	}
}

func (agent *Agent) updatePrices(action, good string, exchange int) {
	if action == "buy" {
		if good == "fruit" {
			if agent.buyFruitPrice < (exchange) {
				agent.buyFruitPrice++
			}
			if agent.buyFruitPrice > (exchange) {
				agent.buyFruitPrice--
			}
		}
		if good == "smoothie" {
			if agent.buySmoothiePrice < (exchange) {
				agent.buySmoothiePrice++
			}
			if agent.buySmoothiePrice > (exchange) {
				agent.buySmoothiePrice--
			}
		}
	}
	if action == "sell" {
		if good == "fruit" {
			if agent.sellFruitPrice < exchange {
				agent.sellFruitPrice++
			}
			if agent.sellFruitPrice > exchange {
				agent.sellFruitPrice--
			}
		}
		if good == "smoothie" {
			if agent.sellSmoothiePrice < exchange {
				agent.sellSmoothiePrice++
			}
			if agent.sellSmoothiePrice > exchange {
				agent.sellSmoothiePrice--
			}
		}
	}
}

func checkTradeConditions(agent *Agent, good string, action string) bool {
	var exchange int
	if good == "fruit" {
		exchange = (agent.buyFruitPrice + tradingPost.sellFruitPrice) / 2
	} else {
		exchange = (agent.buySmoothiePrice + tradingPost.sellSmoothiePrice) / 2
	}

	if action == "buy" {
		if agent.item != "none" {
			logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; inventory not empty\n", agent.id, good))
			return true
		}
		if agent.coins < exchange {
			logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; too few coins %d / %d\n", agent.id, good, agent.coins, exchange))
			return true
		}
		if good == "fruit" {
			if tradingPost.fruit <= 0 {
				logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; post stock empty\n", agent.id, good))
				return true
			}
			if tradingPost.sellFruitPrice > agent.buyFruitPrice {
				logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; sell price exceeds buy price %d / %d\n", agent.id, good, tradingPost.sellFruitPrice, agent.buyFruitPrice))
				agent.updatePrices("buy", "fruit", exchange)
				updateTradePostPrices("sell", "fruit", exchange)
				return true
			}
		}
		if good == "smoothie" {
			if tradingPost.smoothies <= 0 {
				logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; post stock empty\n", agent.id, good))
				return true
			}
			if tradingPost.sellSmoothiePrice > agent.buySmoothiePrice {
				logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; sell price exceeds buy price %d / %d\n", agent.id, good, tradingPost.sellSmoothiePrice, agent.buySmoothiePrice))
				agent.updatePrices("buy", "smoothie", exchange)
				updateTradePostPrices("sell", "smoothie", exchange)
				return true
			}
		}
	}
	if action == "sell" {
		if agent.item != good {
			logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s sale failed; inventory has %s\n", agent.id, good, agent.item))
			return true
		}
		if tradingPost.coins < exchange {
			logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s sale failed; post too few coins %d / %d\n", agent.id, good, tradingPost.coins, exchange))
			return true
		}
		if good == "fruit" {
			if tradingPost.buyFruitPrice < agent.sellFruitPrice {
				logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s sale failed; sell price exceeds buy price %d / %d\n", agent.id, good, agent.sellFruitPrice, tradingPost.buyFruitPrice))
				agent.updatePrices("sell", "fruit", exchange)
				updateTradePostPrices("buy", "fruit", exchange)
				return true
			}
		}
		if good == "smoothie" {
			if tradingPost.buySmoothiePrice < agent.sellSmoothiePrice {
				logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s sale failed; sell price exceeds buy price %d / %d\n", agent.id, good, agent.sellSmoothiePrice, tradingPost.buySmoothiePrice))
				agent.updatePrices("sell", "smoothie", exchange)
				updateTradePostPrices("buy", "smoothie", exchange)
				return true
			}
		}
	}
	return false
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
			agent.attemptExchange("fruit", "buy")
		} else {
			agent.attemptExchange("smoothie", "buy")
		}
	} else if agent.item == "fruit" {
		if rand.Float32() < agent.strategy[8+hungerIndex] {
			agent.makeSmoothie()
		} else {
			agent.attemptExchange("fruit", "sell")
		}
	} else {
		// has a smoothie
		if rand.Float32()-0.3 < agent.strategy[12+hungerIndex] {
			agent.consumeSmoothie()
		} else {
			agent.attemptExchange("smoothie", "sell")
		}
	}
}

func (agent *Agent) growFruit() bool {
	if agent.item != "none" {
		logAgentAction(agent.id, fmt.Sprintf("Agent %d grow fruit failed, inventory not empty", agent.id))
		return true
	}
	logAgentAction(agent.id, fmt.Sprintf("Agent %d growing fruit\n", agent.id))
	time.Sleep(time.Millisecond * time.Duration(10*1000/simulationSpeed))
	agent.item = "fruit"
	agent.numFruitGrowth++

	return false
}

func (agent *Agent) makeSmoothie() bool {

	if agent.item != "fruit" {
		logAgentAction(agent.id, fmt.Sprintf("Agent %d make smoothie failed, inventory needs fruit", agent.id))
		return true
	}
	logAgentAction(agent.id, fmt.Sprintf("Agent %d making smoothie\n", agent.id))
	time.Sleep(time.Millisecond * time.Duration(10*1000/simulationSpeed))
	agent.item = "smoothie"
	agent.numSmoothiesMade++

	return false
}

func (agent *Agent) consumeSmoothie() bool {
	if agent.item != "smoothie" {
		logAgentAction(agent.id, fmt.Sprintf("Agent %d consume smoothie failed, inventory needs smoothie", agent.id))
		return true
	}
	logAgentAction(agent.id, fmt.Sprintf("Agent %d consuming smoothie\n", agent.id))
	time.Sleep(time.Millisecond * time.Duration(1000/simulationSpeed))
	agent.item = "none"
	agent.hunger = max(agent.hunger-1000, 0)
	agent.numSmoothiesConsumed++
	return false
}

// Simulation comment!
func Simulation() {

	simulationSpeed = 100

	runTradingPost()
	tradingPost.buyFruitPrice = rand.IntN(100)
	tradingPost.sellFruitPrice = rand.IntN(100)
	tradingPost.buySmoothiePrice = rand.IntN(100)
	tradingPost.sellSmoothiePrice = rand.IntN(100)
	var pool []*Agent
	for i := range 10 {
		strategy := [16]float32{}
		for j := range 16 {
			strategy[j] = rand.Float32()
		}
		newAgent := Agent{
			id:                i,
			strategy:          strategy,
			item:              "none",
			buyFruitPrice:     rand.IntN(100),
			buySmoothiePrice:  rand.IntN(100),
			sellFruitPrice:    rand.IntN(100),
			sellSmoothiePrice: rand.IntN(100),
			hunger:            0,
			coins:             1000,
			actionCompleted:   make(chan int),
		}
		fmt.Println("Created agent:", newAgent)
		pool = append(pool, &newAgent)
		go func() {
			for {
				newAgent.selectAction()
			}
		}()
	}
	time.Sleep(time.Duration(500/simulationSpeed) * time.Millisecond)

	var initialBuyFruitPrices []int
	var initialSellFruitPrices []int
	for _, a := range pool {
		initialBuyFruitPrices = append(initialBuyFruitPrices, a.buyFruitPrice)
		initialSellFruitPrices = append(initialSellFruitPrices, a.sellFruitPrice)

	}
	slices.Sort(initialBuyFruitPrices)
	slices.Sort(initialSellFruitPrices)

	go func() {
		for i := 0; ; i++ {
			time.Sleep(time.Millisecond * time.Duration(1000/simulationSpeed))
			if i%1000 == 999 {
				evolveAgents(&pool)
				fmt.Println("Agents evolved!")
			}
		}
	}()
	for i := range 10000 {
		printSimulationSummary(pool, initialBuyFruitPrices, initialSellFruitPrices)
		time.Sleep(time.Millisecond * time.Duration(1000/simulationSpeed))
		printAgentSummary(*(pool[0]))
		printAgentSummary(*(pool[1]))
		printAgentSummary(*(pool[2]))
		printAgentSummary(*(pool[3]))
		for _, a := range pool {
			a.hunger += 200
		}
		if i%100 == 99 {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()

		}
	}
}

func (agent *Agent) attemptExchange(good, action string) {
	time.Sleep(time.Millisecond * time.Duration(1000/simulationSpeed))

	logAgentAction(agent.id, fmt.Sprintf("Agent %d attempting %s of %s\n", agent.id, action, good))
	if good == "fruit" {
		if action == "buy" {
			tradingPost.buyFruitQueue <- agent
			<-agent.actionCompleted
		} else {
			tradingPost.sellFruitQueue <- agent
			<-agent.actionCompleted
		}
	} else {
		if action == "buy" {
			tradingPost.buySmoothieQueue <- agent
			<-agent.actionCompleted
		} else {
			tradingPost.sellSmoothieQueue <- agent
			<-agent.actionCompleted
		}
	}
}

func printSimulationSummary(pool []*Agent, initialBuyFruitPrices []int, initialSellFruitPrices []int) {
	hunger := 0
	coinsSum := 0
	var buyFruitPrices []int
	var sellFruitPrices []int
	var buySmoothiePrices []int
	var sellSmoothiePrices []int
	var allCoins []int
	var hungers []int
	for _, a := range pool {
		hunger += a.hunger
		coinsSum += a.coins
		allCoins = append(allCoins, a.coins)
		buyFruitPrices = append(buyFruitPrices, a.buyFruitPrice)
		sellFruitPrices = append(sellFruitPrices, a.sellFruitPrice)
		buySmoothiePrices = append(buySmoothiePrices, a.buySmoothiePrice)
		sellSmoothiePrices = append(sellSmoothiePrices, a.sellSmoothiePrice)
		hungers = append(hungers, a.hunger)

	}

	slices.Sort(buyFruitPrices)
	fmt.Println("fruit buys:", buyFruitPrices, tradingPost.sellFruitPrice)
	slices.Sort(sellFruitPrices)
	fmt.Println("fruit sells:", sellFruitPrices, tradingPost.buyFruitPrice)
	fmt.Println("smoothie buys:", buySmoothiePrices, tradingPost.sellSmoothiePrice)
	fmt.Println("smoothie sells:", sellSmoothiePrices, tradingPost.buySmoothiePrice)
	//slices.Sort(allCoins)
	fmt.Println("coins:", allCoins, coinsSum)
	fmt.Println("hunger:", hungers)

	fmt.Println(hunger, tradingPost.coins, tradingPost.fruit, tradingPost.smoothies, "total hunger, post coins, fruit, smoothies")

}

func printAgentSummary(agent Agent) {
	fmt.Println(agent.numFruitBuys+agent.numFruitSales+agent.numSmoothieBuys+agent.numSmoothieSales+agent.numFruitGrowth+agent.numSmoothiesMade+agent.numSmoothiesConsumed, "[", agent.numFruitBuys, agent.buyFruitPrice, "], [", agent.numFruitSales, agent.sellFruitPrice, "]", agent.numSmoothieBuys, agent.numSmoothieSales, agent.numFruitGrowth, agent.numSmoothiesMade, agent.numSmoothiesConsumed)
}

func logAgentAction(id int, message string) {
	if false && (id == 0 || id == -1) {
		fmt.Print(message)
	}
}
func logTradeFailure(id int, message string) {
	if false && (id == 0 || id == -1) {
		fmt.Print(message)
	}
}

func evolveAgents(pool *[]*Agent) {

	sort.Slice(*pool, func(i, j int) bool {
		return (*pool)[i].coins /*-(*pool)[i].hunger */ < (*pool)[j].coins /*-(*pool)[j].hunger */
	})

	//for i := 0; i < (len(*pool)); i++ {
	//	if (*pool)[i].hunger > 10000 {
	//*pool = removeAgent((*pool), i)
	// }
	//}
	deathIndex := rand.IntN(3)
	copiedIndex := rand.IntN(3) + 7
	(*pool)[deathIndex].strategy = (*pool)[copiedIndex].strategy
	(*pool)[deathIndex].item = (*pool)[copiedIndex].item
	(*pool)[deathIndex].buyFruitPrice = (*pool)[copiedIndex].buyFruitPrice
	(*pool)[deathIndex].buySmoothiePrice = (*pool)[copiedIndex].buySmoothiePrice
	(*pool)[deathIndex].sellFruitPrice = (*pool)[copiedIndex].sellFruitPrice
	(*pool)[deathIndex].sellSmoothiePrice = (*pool)[copiedIndex].sellSmoothiePrice
	// (*pool)[0].hunger = 0
	//coins:             1000,
	//actionCompleted:   make(chan int),
}

func removeAgent(pool []*Agent, index int) []*Agent {
	ret := make([]*Agent, 0)
	ret = append(ret, pool[:index]...)
	return append(ret, pool[index+1:]...)
}
