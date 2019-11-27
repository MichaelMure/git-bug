# Emoji
Emoji is a simple golang package.

[![wercker status](https://app.wercker.com/status/7bef60de2c6d3e0e6c13d56b2393c5d8/s/master "wercker status")](https://app.wercker.com/project/byKey/7bef60de2c6d3e0e6c13d56b2393c5d8)
[![Coverage Status](https://coveralls.io/repos/kyokomi/emoji/badge.png?branch=master)](https://coveralls.io/r/kyokomi/emoji?branch=master)
[![GoDoc](https://godoc.org/github.com/kyokomi/emoji?status.svg)](https://godoc.org/github.com/kyokomi/emoji)

Get it:

```
go get github.com/kyokomi/emoji
```

Import it:

```
import (
	"github.com/kyokomi/emoji"
)
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/kyokomi/emoji"
)

func main() {
	fmt.Println("Hello World Emoji!")

	emoji.Println(":beer: Beer!!!")

	pizzaMessage := emoji.Sprint("I like a :pizza: and :sushi:!!")
	fmt.Println(pizzaMessage)
}
```

## Demo

![demo](screen/image.png)

## Reference

- [GitHub EMOJI CHEAT SHEET](http://www.emoji-cheat-sheet.com/)

## License

[MIT](https://github.com/kyokomi/emoji/blob/master/LICENSE)
