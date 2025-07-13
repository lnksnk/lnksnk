package globallisten

import "github.com/lnksnk/lnksnk/listening"

var LISTEN listening.LISTENING

func init() {
	LISTEN = listening.NewListen()
}
