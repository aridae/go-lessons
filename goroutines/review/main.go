package main

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// You need to review this code. Don't change this code, you can leave comments right here.

// (1) общий коммент: если канал необходимо передать в метод, лучше указать,
// что данные в канал data могут в функции только записываться: data chan<- string,
// чтобы было ясно, что это канал для выходных данных, а не для входных
//
// (2) тут было бы проще отдавать в ответе status string,
// тк чистую функцию проще тестить, а записывать в выходной канал в функции-обертке
// + так будет меньше возможностей выстрелить себе в ногу, потому что
// если канал вне функции закроют, мы запаникуем и упадем
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
	// (3) буфер канала даст выйгрыш только если
	// горутины-сендеры после записи значения выполняли бы
	// какую-то еще полезную работу (и не блокировались бы на канале)
	data := make(chan string, 100)

	// продолжение (3): здесь итерация по каналу в мейн горутине не дает больше профита, чем просто итерация по слайсу
	// профит был бы, если бы в нескольких горутинах мы итерировались по каналу и выполняли полезную работу, пока его не закроют
	for source := range fetchSources() {
		// в продолжение комментария (2): риск паники и нет рекавера,
		// в случае паники упадет все приложение
		go fetchDataWrong(source, data)
	}

	// (4) итерируемся по каналу пока он не будет закрыт
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
