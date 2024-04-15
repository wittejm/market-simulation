package smoothies

import (
	"fmt"
)

func (agent *Agent) attemptBuyOld(good string) bool {
	if agent.item != "none" {
		fmt.Printf("Agent %d inv not empty; buy failed\n", agent.id)
		return false
	}
	var sell int
	var buy int
	waitChannel := make(chan int)

	//go delayedPing(waitChannel, 20)
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
			agent.buyFruitPrice--
		} else {
			agent.buySmoothiePrice--
		}
		return true
	}

	//fmt.Printf("Agent %d buy failed, sale price too high\n", agent.id)
	if good == "fruit" {
		agent.buyFruitPrice++
	} else {
		agent.buySmoothiePrice++
	}
	return false

}

func (agent *Agent) attemptSellOld(good string) bool {
	if agent.item == "none" {
		fmt.Printf("Agent %d inv empty; sell failed\n", agent.id)
		return false
	}
	var buy int
	var sell int
	waitChannel := make(chan int)
	//go delayedPing(waitChannel, 100)

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
			agent.sellFruitPrice++
		} else {
			agent.sellSmoothiePrice++
		}
		exchange := (sell + buy) / 2
		agent.item = "none"
		agent.coins += exchange
		fmt.Printf("Agent %d sold %s for %d\n", agent.id, good, exchange)
		return true

	}
	//fmt.Printf("Agent %d sale failed, buy price too low\n", agent.id)
	if good == "fruit" {
		agent.sellFruitPrice--
	} else {
		agent.sellSmoothiePrice--
	}
	return false

}
