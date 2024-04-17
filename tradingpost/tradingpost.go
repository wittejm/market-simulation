package tradingpost

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"sort"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

/*
This simulation's characteristics:
	* Agents can: buy/sell fruit/smoothies, grow fruit, make smoothies, consume smoothies
	* Random initialization of strategy vector
	* prices move toward the buy/sell's average value after every attempted buy/sell.
	* evolution function changes the losing agent's strategy and prices to equal the winning agent's
	* evolution function can either measure coins or coins minus hunger

This simulation's behaviors when learning on a coin measure:
	* Prices stabilize to an average
	* Agents learn various sell strategies
	* Fruit price goes to 1
	* Agents learn to buy fruit, turn it into smoothies and sell them
	* Smoothie price goes to 1
	* Agents learn to grow fruit and sell it for a small profit
 *

*/

func pointsFromArray(arr []int) plotter.XYs {
	pts := make(plotter.XYs, len(arr))
	for i := range pts {
		pts[i].X = float64(i)
		pts[i].Y = float64(arr[i])
	}
	return pts
}

func plotResult(pool []*Agent) {

	p := plot.New()

	p.Title.Text = "Simulation"
	p.X.Label.Text = "evolve steps"
	p.Y.Label.Text = "Y"
	p2 := plot.New()

	p2.Title.Text = "Hunger"
	p2.X.Label.Text = "evolve steps"
	p2.Y.Label.Text = "Y"
	fmt.Println("allFruitBuyPrice", allFruitBuyPrice)
	fmt.Println("allSmoothieBuyPrice", allSmoothieBuyPrice)
	fmt.Println("allNumFruit", allNumFruit)
	fmt.Println("allNumSmoothies", allNumSmoothies)
	var ages []int
	for _, a := range pool {
		ages = append(ages, a.age)
	}
	fmt.Println("ages", ages)
	//fmt.Println("allHunger", allHunger)

	err := plotutil.AddLinePoints(p,
		"Fruit Price", pointsFromArray(allFruitBuyPrice),
		"Smoothie Price", pointsFromArray(allSmoothieBuyPrice),
		"All Fruit Count", pointsFromArray(allNumFruit),
		"All Smoothie Count", pointsFromArray(allNumSmoothies))
	if err != nil {
		panic(err)
	}
	err2 := plotutil.AddLinePoints(p2,
		"All Hunger", pointsFromArray(allHunger))
	if err2 != nil {
		panic(err2)
	}

	// Save the plot to a PNG file.
	if err := p.Save(4*vg.Inch, 4*vg.Inch, "evolve.png"); err != nil {
		panic(err)
	}
	if err := p2.Save(4*vg.Inch, 4*vg.Inch, "hunger.png"); err != nil {
		panic(err)
	}
}

// randomPoints returns some random x, y points.
func randomPoints(n int) plotter.XYs {
	pts := make(plotter.XYs, n)
	for i := range pts {
		if i == 0 {
			pts[i].X = rand.Float64()
		} else {
			pts[i].X = pts[i-1].X + rand.Float64()
		}
		pts[i].Y = pts[i].X + 10*rand.Float64()
	}
	return pts
}

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
	previousCoins        int
	actionCompleted      chan bool
	numFruitBuys         int
	numFruitSales        int
	numSmoothieBuys      int
	numSmoothieSales     int
	numFruitGrowth       int
	numSmoothiesMade     int
	numSmoothiesConsumed int
	age                  int
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
	mu                sync.Mutex
}

var tradingPost TradingPost
var simulationSpeed int // a multiplier

func getPostPricePointer(action, good string) *int {
	if good == "fruit" {
		if action == "buy" {
			return &tradingPost.buyFruitPrice
		}
		if action == "sell" {
			return &tradingPost.sellFruitPrice
		}
	}
	if good == "smoothie" {
		if action == "buy" {
			return &tradingPost.buySmoothiePrice
		}
		if action == "sell" {
			return &tradingPost.sellSmoothiePrice
		}
	}
	return nil
}

func getAgentPricePointer(agent *Agent, action, good string) *int {
	if good == "fruit" {
		if action == "buy" {
			return &agent.buyFruitPrice
		}
		if action == "sell" {
			return &agent.sellFruitPrice
		}
	}
	if good == "smoothie" {
		if action == "buy" {
			return &agent.buySmoothiePrice
		}
		if action == "sell" {
			return &agent.sellSmoothiePrice
		}
	}
	return nil
}

func getAgentCounterPointer(agent *Agent, action, good string) *int {
	if action == "buy" {
		if good == "fruit" {
			return &agent.numFruitBuys
		}
		if good == "smoothie" {
			return &agent.numSmoothieBuys
		}
	}
	if action == "sell" {
		if good == "fruit" {
			return &agent.numFruitSales
		}
		if good == "smoothie" {
			return &agent.numSmoothieSales
		}
	}
	return nil
}

func getCoinsPointer(agent *Agent, actorType string) *int {
	if actorType == "agent" {
		return &agent.coins
	}
	if actorType == "post" {
		return &tradingPost.coins
	}
	return nil
}
func getOtherItem(items []string, item string) string {
	if item == items[0] {
		return items[1]
	}
	if item == items[1] {
		return items[0]
	}
	return ""
}

func getNextClientQueue(action, good string) chan *Agent {
	if action == "buy" {
		if good == "fruit" {
			return tradingPost.buyFruitQueue
		}
		if good == "smoothie" {
			return tradingPost.buySmoothieQueue
		}
	}
	if action == "sell" {
		if good == "fruit" {
			return tradingPost.sellFruitQueue
		}
		if good == "smoothie" {
			return tradingPost.sellSmoothieQueue
		}
	}
	return nil
}

func tradeLoop(action, good string) {
	for {
		nextClientQueue := getNextClientQueue(action, good)
		nextClient := <-nextClientQueue
		tradingPost.mu.Lock()
		logAgentAction(nextClient.id, fmt.Sprintf("initiated trade: Agent %d %s %s\n", nextClient.id, action, good))
		failure := checkTradeConditions(nextClient, action, good)
		if failure {
			logTradeFailure(nextClient.id, fmt.Sprintf("trade failure %s %s\n", action, good))
			nextClient.actionCompleted <- true
			tradingPost.mu.Unlock()
			continue
		}
		postAction := getOtherItem([]string{"buy", "sell"}, action)
		clientPrice := *getAgentPricePointer(nextClient, action, good)
		postPrice := *getPostPricePointer(postAction, good)

		exchange := (clientPrice + postPrice) / 2
		postCoins := getCoinsPointer(nil, "post")
		agentCoins := getCoinsPointer(nextClient, "agent")
		postInventory := getPostInventory(good)
		if action == "buy" {
			*postCoins += exchange
			*agentCoins -= exchange
			nextClient.item = good
			*postInventory--
		}
		if action == "sell" {
			*postCoins -= exchange
			*agentCoins += exchange
			nextClient.item = "none"
			*postInventory++
		}
		logAgentAction(nextClient.id, fmt.Sprintf("Agent %d completed %s %s for %d. coins: %d\n", nextClient.id, action, good, exchange, nextClient.coins))
		updateTradePostPrices(nextClient.id, action, good, exchange)
		nextClient.updateAgentPrice(action, good, exchange)
		if *postInventory < 50 {
			//logTradeFailure(nextClient.id, fmt.Sprintf("Low post inventory; increasing %s %s price\n", action, good))
			updateTradePostPrices(nextClient.id, action, good, 1000)
		}
		if *postInventory > 150 {
			//logTradeFailure(nextClient.id, fmt.Sprintf("High post inventory; decreasing %s %s price\n", action, good))
			updateTradePostPrices(nextClient.id, action, good, 1)
		}

		nextClient.actionCompleted <- false
		tradingPost.mu.Unlock()

	}

}

func getPostInventory(good string) *int {
	if good == "fruit" {
		return &tradingPost.fruit
	}
	if good == "smoothie" {
		return &tradingPost.smoothies
	}
	return nil
}
func runTradingPost() {
	tradingPost.coins = 1000000
	tradingPost.fruit = 100
	tradingPost.smoothies = 100

	tradingPost.buyFruitPrice = rand.IntN(100)
	tradingPost.sellFruitPrice = rand.IntN(100)
	tradingPost.buySmoothiePrice = rand.IntN(100)
	tradingPost.sellSmoothiePrice = rand.IntN(100)

	tradingPost.buyFruitQueue = make(chan *Agent)
	tradingPost.sellFruitQueue = make(chan *Agent)
	tradingPost.buySmoothieQueue = make(chan *Agent)
	tradingPost.sellSmoothieQueue = make(chan *Agent)

	go tradeLoop("buy", "fruit")  // Selling fruit to buyers
	go tradeLoop("sell", "fruit") // etc.
	go tradeLoop("buy", "smoothie")
	go tradeLoop("sell", "smoothie")

}

func updateTradePostPrices(agentId int, action, good string, exchange int) {
	pricePointer := getPostPricePointer(action, good)
	if *pricePointer < exchange {
		*pricePointer++
		logTradeFailure(agentId, fmt.Sprintf("post price below exchange: %s %s price increase\n", action, good))
	}
	if *pricePointer > exchange {
		*pricePointer--
		logTradeFailure(agentId, fmt.Sprintf("post price above exchange:  %s %s price decrease\n", action, good))
	}
}

func (agent *Agent) updateAgentPrice(action, good string, exchange int) {
	pricePointer := getAgentPricePointer(agent, action, good)
	if *pricePointer < exchange {
		*pricePointer++
	}
	if *pricePointer > exchange {
		*pricePointer--
	}
}

func checkTradeConditions(agent *Agent, action, good string) bool {
	postAction := getOtherItem([]string{"buy", "sell"}, action)
	postPrice := getPostPricePointer(postAction, good)
	agentPrice := getAgentPricePointer(agent, action, good)
	exchange := (*postPrice + *agentPrice) / 2

	if action == "buy" {
		if agent.item != "none" {
			logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; inventory not empty\n", agent.id, good))
			return true
		}
		if *agentPrice < *postPrice {
			// agent won't spend more than its buy price.
			logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; post sell price exceeds agent buy price %d / %d\n", agent.id, good, *postPrice, *agentPrice))
			*agentPrice++
			*postPrice--
			return true
		}
		if agent.coins < exchange {
			logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; too few coins %d / %d\n", agent.id, good, agent.coins, exchange))
			return true
		}
		inventory := *getPostInventory(good)
		if inventory <= 0 {
			logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; post stock empty\n", agent.id, good))
			return true
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
		if *agentPrice > *postPrice {
			// agent won't sell for less than its sale price.
			logTradeFailure(agent.id, fmt.Sprintf("Agent %d %s purchase failed; agent sell price exceeds post buy price %d / %d\n", agent.id, good, *agentPrice, *postPrice))
			*agentPrice--
			*postPrice++
			return true
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
			agent.attemptExchange("buy", "fruit")
		} else {
			agent.attemptExchange("buy", "smoothie")
		}
	} else if agent.item == "fruit" {
		if rand.Float32() < agent.strategy[8+hungerIndex] {
			agent.makeSmoothie()
		} else {
			agent.attemptExchange("sell", "fruit")
		}
	} else {
		// has a smoothie
		if rand.Float32()-0.3 < agent.strategy[12+hungerIndex] {
			agent.consumeSmoothie()
		} else {
			//agent.consumeSmoothie()

			agent.attemptExchange("sell", "smoothie")
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
	agent.hunger = max(agent.hunger-10000, 0)
	agent.numSmoothiesConsumed++
	return false
}

var allFruitBuyPrice []int
var allSmoothieBuyPrice []int
var allNumFruit []int
var allNumSmoothies []int
var allHunger []int

// Simulation comment!
func Simulation() {

	simulationSpeed = 1000

	runTradingPost()

	var pool []*Agent
	for i := range 30 {
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
			previousCoins:     1000,
			actionCompleted:   make(chan bool),
			age:               0,
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
	go func() {
		for i := 0; ; i++ {
			time.Sleep(time.Millisecond * time.Duration(1000/simulationSpeed))
			if i%1000 == 999 {
				allFruitBuyPrice = append(allFruitBuyPrice, min(300, tradingPost.buyFruitPrice))
				allSmoothieBuyPrice = append(allSmoothieBuyPrice, min(300, tradingPost.buySmoothiePrice))
				allNumFruit = append(allNumFruit, min(300, tradingPost.fruit))
				allNumSmoothies = append(allNumSmoothies, min(300, tradingPost.smoothies))
				hunger := 0
				for _, a := range pool {
					hunger += a.hunger
				}
				allHunger = append(allHunger, hunger)
				evolveAgents(&pool)
				plotResult(pool)
				fmt.Println("Agents evolved!")
				//printAgentSummary("Poorest agent", *(pool[1]))
				//printAgentSummary("Richest agent", *(pool[9]))

			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Millisecond * time.Duration(1000/simulationSpeed))
			for _, a := range pool {
				a.hunger += 200
			}
		}
	}()

	for {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		printSimulationSummary(pool)
		time.Sleep(time.Millisecond * time.Duration(1000/simulationSpeed))
		//printAgentSummary("Poorest agent", *(pool[0]))
		//printAgentSummary("Richest agent", *(pool[9]))

	}
}

func (agent *Agent) attemptExchange(action, good string) {
	time.Sleep(time.Millisecond * time.Duration(1000/simulationSpeed))

	logAgentAction(agent.id, fmt.Sprintf("Agent %d attempting %s of %s\n", agent.id, action, good))
	clientChan := getNextClientQueue(action, good)
	clientChan <- agent
	failure := <-agent.actionCompleted
	if !failure {
		counter := getAgentCounterPointer(agent, action, good)
		*counter++
	}
}

func printSimulationSummary(pool []*Agent) {
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

func printAgentSummary(message string, agent Agent) {
	fmt.Println(message, agent.numFruitBuys+agent.numFruitSales+agent.numSmoothieBuys+agent.numSmoothieSales+agent.numFruitGrowth+agent.numSmoothiesMade+agent.numSmoothiesConsumed, "f b/pr:[", agent.numFruitBuys, agent.buyFruitPrice, "], f s/pr:[", agent.numFruitSales, agent.sellFruitPrice, "], s b/pr:[", agent.numSmoothieBuys, agent.buySmoothiePrice, "], s s/pr:[", agent.numSmoothieSales, agent.sellSmoothiePrice, "]", agent.numFruitGrowth, agent.numSmoothiesMade, agent.numSmoothiesConsumed)
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
		//return (*pool)[i].coins-(*pool)[i].previousCoins < (*pool)[j].coins-(*pool)[j].previousCoins
		return (*pool)[i].coins-(*pool)[i].hunger < (*pool)[j].coins-(*pool)[j].hunger
	})
	for _, a := range *pool {
		a.previousCoins = a.coins
		a.age += 1
	}

	//for i := 0; i < (len(*pool)); i++ {
	//	if (*pool)[i].hunger > 10000 {
	//*pool = removeAgent((*pool), i)
	// }
	//}
	type AgentMetrics struct {
		lastFruitBuys         int
		lastFruitSales        int
		lastSmoothieBuys      int
		lastSmoothieSales     int
		lastFruitGrowth       int
		lastSmoothiesMade     int
		lastSmoothiesConsumed int
	}
	var metrics []AgentMetrics = make([]AgentMetrics, len(*pool))
	for _, a := range *pool {
		metrics[a.id] = AgentMetrics{
			lastFruitBuys:         a.numFruitBuys - metrics[a.id].lastFruitBuys,
			lastFruitSales:        a.numFruitSales - metrics[a.id].lastFruitSales,
			lastSmoothieBuys:      a.numSmoothieBuys - metrics[a.id].lastSmoothieBuys,
			lastSmoothieSales:     a.numSmoothieSales - metrics[a.id].lastSmoothieSales,
			lastFruitGrowth:       a.numFruitGrowth - metrics[a.id].lastFruitGrowth,
			lastSmoothiesMade:     a.numSmoothiesMade - metrics[a.id].lastSmoothiesMade,
			lastSmoothiesConsumed: a.numSmoothiesConsumed - metrics[a.id].lastSmoothiesConsumed,
		}
	}

	fmt.Println("Poorest", metrics[(*pool)[0].id])
	fmt.Println("Richest", metrics[(*pool)[len(*pool)-1].id])
	deathIndex := 0               //rand.IntN(3)
	copiedIndex := len(*pool) - 1 //rand.IntN(3) + 7
	(*pool)[deathIndex].strategy = (*pool)[copiedIndex].strategy
	(*pool)[deathIndex].item = (*pool)[copiedIndex].item
	(*pool)[deathIndex].buyFruitPrice = (*pool)[copiedIndex].buyFruitPrice
	(*pool)[deathIndex].buySmoothiePrice = (*pool)[copiedIndex].buySmoothiePrice
	(*pool)[deathIndex].sellFruitPrice = (*pool)[copiedIndex].sellFruitPrice
	(*pool)[deathIndex].sellSmoothiePrice = (*pool)[copiedIndex].sellSmoothiePrice
	(*pool)[deathIndex].age = 0

	for i := range 2 {
		randomizedIndex := i + 1
		for j := range 16 {
			(*pool)[randomizedIndex].strategy[j] = rand.Float32()
		}

		(*pool)[randomizedIndex].item = "none"
		(*pool)[randomizedIndex].buyFruitPrice = rand.IntN(100)
		(*pool)[randomizedIndex].buySmoothiePrice = rand.IntN(100)
		(*pool)[randomizedIndex].sellFruitPrice = rand.IntN(100)
		(*pool)[randomizedIndex].sellSmoothiePrice = rand.IntN(100)
		(*pool)[randomizedIndex].age = 0

	}

}
