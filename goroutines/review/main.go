package main

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// You need to review this code. Don't change this code, you can leave comments right here.

func fetchDataWrong(source string, data chan string) {
	// Random delay from 0 to 1 second.
	time.Sleep(time.Millisecond * rand.N[time.Duration](1000))

	// Generates an error 20% of the time.
	if rand.N[int](10) < 2 {
		data <- fmt.Sprintf("failed to fetch data from %s", source)
		return
	}

	data <- fmt.Sprintf("data from %s", source)
}

func main() {
	data := make(chan string, 100)

	for source := range fetchSources() {
		go fetchDataWrong(source, data)
	}

	// итерируемся по каналу пока он не будет закрыт
	// канал data не закрыт -> залочим основную горутину на этом месте
	// канал нужно закрыть, когда новые данные перестанут поступать,
	// но основная горутина не знает, когда все сурсы уже опрошены (считая, что количество сурсов не фиксировано)
	// предложение тут применить sync.WaitGroup, дождаться когда все ресурсы будут опрошены и закрыть канал data
	// выводить статусы можно не блокируясь в отдельной горутине
	for status := range data {
		fmt.Println(status)
	}
}

func fetchSources() <-chan string {
	sources := make(chan string)

	go func() {
		for _, s := range []string{"Source1", "Source2", "Source3", "Source4", "Source5"} {
			sources <- s
		}
		close(sources)
	}()

	return sources
}
