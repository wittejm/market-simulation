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
	fmt.Println("allFruitBuyPrice", allBuyPrices["fruit"])
	fmt.Println("allSmoothieBuyPrice", allBuyPrices["smoothie"])
	fmt.Println("allNumFruit", allNumInventory["fruit"])
	fmt.Println("allNumSmoothies", allNumInventory["smoothie"])
	var ages []int
	for _, a := range pool {
		ages = append(ages, a.age)
	}
	fmt.Println("ages", ages)
	//fmt.Println("allHunger", allHunger)

	err := plotutil.AddLinePoints(p,
		"Fruit Price", pointsFromArray(allBuyPrices["fruit"]),
		"Smoothie Price", pointsFromArray(allBuyPrices["smoothie"]),
		"All Fruit Count", pointsFromArray(allNumInventory["fruit"]),
		"All Smoothie Count", pointsFromArray(allNumInventory["smoothie"]))
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

// Agent comment
type Agent struct {
	id              int
	strategy        [16]float32
	item            string
	buyPrices       map[string]*int
	sellPrices      map[string]*int
	numActions      map[string]*int
	hunger          int
	coins           int
	previousCoins   int
	actionCompleted chan bool
	age             int
}

// The TradingPost comment
type TradingPost struct {
	coins      int
	inventory  map[string]*int
	buyPrices  map[string]*int
	sellPrices map[string]*int
	queues     map[string]*chan *Agent
	mu         sync.Mutex
}

var tradingPost TradingPost
var simulationSpeed int // a multiplier

func getPostPricePointer(action, good string) *int {
	if action == "buy" {
		return tradingPost.buyPrices[good]
	}
	if action == "sell" {
		return tradingPost.sellPrices[good]
	}
	return nil
}

func getAgentPricePointer(agent *Agent, action, good string) *int {
	if action == "buy" {
		return agent.buyPrices[good]
	}
	if action == "sell" {
		return agent.sellPrices[good]
	}
	return nil
}

func getAgentCounterPointer(agent *Agent, action, good string) *int {
	if action == "buy" {
		return agent.numActions[fmt.Sprintf("buy %s", good)]
	}
	if action == "sell" {
		return agent.numActions[fmt.Sprintf("sell %s", good)]
	}
	return agent.numActions[action] // for "forage", "grow", "make"
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
func getOtherArrayElement(items []string, item string) string {
	if item == items[0] {
		return items[1]
	}
	if item == items[1] {
		return items[0]
	}
	return ""
}

func getNextClientQueue(action, good string) chan *Agent {
	return *tradingPost.queues[fmt.Sprintf("%s %s", action, good)]
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
		postAction := getOtherArrayElement([]string{"buy", "sell"}, action)
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
	return tradingPost.inventory[good]
}

func runTradingPost() {
	tradingPost.coins = 1000000
	tradingPost.inventory = make(map[string]*int)
	tradingPost.buyPrices = make(map[string]*int)
	tradingPost.sellPrices = make(map[string]*int)
	tradingPost.queues = make(map[string]*chan *Agent)

	for _, good := range goods {
		tradingPost.inventory[good] = func() *int { v := 100; return &v }()
		tradingPost.buyPrices[good] = func() *int { v := rand.IntN(100); return &v }()
		tradingPost.sellPrices[good] = func() *int { v := rand.IntN(100); return &v }()
		tradingPost.queues[fmt.Sprintf("buy %s", good)] = func() *chan *Agent { v := make(chan *Agent); return &v }()
		tradingPost.queues[fmt.Sprintf("sell %s", good)] = func() *chan *Agent { v := make(chan *Agent); return &v }()
		go tradeLoop("buy", good)  // Selling to buyers
		go tradeLoop("sell", good) // Buying from sellers
	}
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
	postAction := getOtherArrayElement([]string{"buy", "sell"}, action)
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
	*agent.numActions["grow"]++

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
	*agent.numActions["make"]++

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
	*agent.numActions["consume"]++
	return false
}

var allBuyPrices map[string][]int
var allNumInventory map[string][]int
var allHunger []int
var goods []string

// Simulation comment!
func Simulation() {

	simulationSpeed = 1000

	goods = []string{"seed", "fruit", "smoothie"}

	allBuyPrices = make(map[string][]int)
	allNumInventory = make(map[string][]int)
	allHunger = make([]int, 0)

	for _, good := range goods {
		allBuyPrices[good] = make([]int, 0)
		allNumInventory[good] = make([]int, 0)
	}
	runTradingPost()

	var pool []*Agent
	for i := range 30 {
		strategy := [16]float32{}
		for j := range 16 {
			strategy[j] = rand.Float32()
		}
		buyPrices := make(map[string]*int)
		sellPrices := make(map[string]*int)
		numActions := make(map[string]*int)
		for _, good := range goods {
			buyPrices[good] = func() *int { v := rand.IntN(100); return &v }()
			sellPrices[good] = func() *int { v := rand.IntN(100); return &v }()
			numActions[fmt.Sprintf("buy %s", good)] = func() *int { v := 0; return &v }()
			numActions[fmt.Sprintf("sell %s", good)] = func() *int { v := 0; return &v }()
		}
		numActions["grow"] = func() *int { v := 0; return &v }()
		numActions["make"] = func() *int { v := 0; return &v }()
		numActions["consume"] = func() *int { v := 0; return &v }()

		newAgent := Agent{
			id:              i,
			strategy:        strategy,
			item:            "none",
			buyPrices:       buyPrices,
			sellPrices:      sellPrices,
			hunger:          0,
			coins:           1000,
			previousCoins:   1000,
			actionCompleted: make(chan bool),
			age:             0,
			numActions:      numActions,
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
				for _, good := range goods {
					allBuyPrices[good] = append(allBuyPrices[good], min(300, *tradingPost.buyPrices[good]))
					allNumInventory[good] = append(allNumInventory[good], min(300, *tradingPost.inventory[good]))
				}
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
	buyPrices := make(map[string][]int)
	sellPrices := make(map[string][]int)
	for _, good := range goods {
		buyPrices[good] = make([]int, 0)
		sellPrices[good] = make([]int, 0)
	}
	var allCoins []int
	var hungers []int
	for _, a := range pool {
		hunger += a.hunger
		coinsSum += a.coins
		allCoins = append(allCoins, a.coins)
		for _, good := range goods {
			buyPrices[good] = append(buyPrices[good], *a.buyPrices[good])
			sellPrices[good] = append(sellPrices[good], *a.sellPrices[good])
		}
		hungers = append(hungers, a.hunger)
	}
	for _, good := range goods {
		slices.Sort(buyPrices[good])
		fmt.Printf("%s buys: %v %v", good, buyPrices[good], tradingPost.sellPrices[good])
	}

	fmt.Println("coins:", allCoins, coinsSum)
	fmt.Println("hunger:", hungers)

	fmt.Println(hunger, tradingPost.coins, tradingPost.inventory["fruit"], tradingPost.inventory["smoothie"], "total hunger, post coins, fruit, smoothies")

}

func printAgentSummary(message string, agent Agent) {
	fmt.Println(message, "f b/pr:[", agent.numActions["buy fruit"], agent.buyPrices["fruit"], "], f s/pr:[", agent.numActions["sell fruit"], agent.sellPrices["fruit"], "], s b/pr:[", agent.numActions["buy smoothie"], agent.buyPrices["smoothie"], "], s s/pr:[", agent.numActions["sell smoothie"], agent.sellPrices["smoothie"], "]", agent.numActions["grow"], agent.numActions["make"], agent.numActions["consume"])
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
		a.age++
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
			lastFruitBuys:         *a.numActions["buy fruit"] - metrics[a.id].lastFruitBuys,
			lastFruitSales:        *a.numActions["sell fruit"] - metrics[a.id].lastFruitSales,
			lastSmoothieBuys:      *a.numActions["buy smoothie"] - metrics[a.id].lastSmoothieBuys,
			lastSmoothieSales:     *a.numActions["sell smoothie"] - metrics[a.id].lastSmoothieSales,
			lastFruitGrowth:       *a.numActions["grow"] - metrics[a.id].lastFruitGrowth,
			lastSmoothiesMade:     *a.numActions["make"] - metrics[a.id].lastSmoothiesMade,
			lastSmoothiesConsumed: *a.numActions["consume"] - metrics[a.id].lastSmoothiesConsumed,
		}
	}

	fmt.Println("Poorest", metrics[(*pool)[0].id])
	fmt.Println("Richest", metrics[(*pool)[len(*pool)-1].id])
	deathIndex := 0               //rand.IntN(3)
	copiedIndex := len(*pool) - 1 //rand.IntN(3) + 7
	(*pool)[deathIndex].strategy = (*pool)[copiedIndex].strategy
	(*pool)[deathIndex].item = (*pool)[copiedIndex].item
	for _, good := range goods {
		*(*pool)[deathIndex].buyPrices[good] = *(*pool)[copiedIndex].buyPrices[good]
		*(*pool)[deathIndex].sellPrices[good] = *(*pool)[copiedIndex].sellPrices[good]
	}
	(*pool)[deathIndex].age = 0

	for i := range 2 {
		randomizedIndex := i + 1
		for j := range 16 {
			(*pool)[randomizedIndex].strategy[j] = rand.Float32()
		}

		(*pool)[randomizedIndex].item = "none"
		for _, good := range goods {
			*(*pool)[deathIndex].buyPrices[good] = rand.IntN(100)
			*(*pool)[deathIndex].sellPrices[good] = rand.IntN(100)
		}
		(*pool)[randomizedIndex].age = 0
	}
}
