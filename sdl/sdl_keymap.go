package sdlw

import (
	"github.com/skelterjohn/go.wde"
)

var keys = []string {
	"",
	"",
	"",
	"",
	wde.KeyA, //4
	wde.KeyB,
	wde.KeyC,
	wde.KeyD,
	wde.KeyE,
	wde.KeyF,
	wde.KeyG,
	wde.KeyH,
	wde.KeyI,
	wde.KeyJ,
	wde.KeyK,
	wde.KeyL,
	wde.KeyM,
	wde.KeyN,
	wde.KeyO,
	wde.KeyP,
	wde.KeyQ,
	wde.KeyR,
	wde.KeyS,
	wde.KeyT,
	wde.KeyU,
	wde.KeyV,
	wde.KeyW,
	wde.KeyX,
	wde.KeyY,
	wde.KeyZ,
	wde.Key1,
	wde.Key2,
	wde.Key3,
	wde.Key4,
	wde.Key5,
	wde.Key6,
	wde.Key7,
	wde.Key8,
	wde.Key9,
	wde.Key0,
	wde.KeyReturn,
	wde.KeyEscape,
	wde.KeyBackspace,
	wde.KeyTab,
	wde.KeySpace,
	wde.KeyMinus,
	wde.KeyEqual,
	wde.KeyLeftBracket,
	wde.KeyRightBracket,
	wde.KeyBackslash,
	"", //50
	wde.KeySemicolon,
	wde.KeyQuote,
	wde.KeyBackTick,
	wde.KeyComma,
	wde.KeyPeriod,
	wde.KeySlash,
	"",
	wde.KeyF1,
	wde.KeyF2,
	wde.KeyF3,
	wde.KeyF4,
	wde.KeyF5,
	wde.KeyF6,
	wde.KeyF7,
	wde.KeyF8,
	wde.KeyF9,
	wde.KeyF10,
	wde.KeyF11,
	wde.KeyF12,
	"", //70
	"",
	"",
	wde.KeyInsert,
	"",
	"",
	wde.KeyDelete,
}

func init() {
	keys = append(keys, make([]string, 256 - len(keys))...)
	keys[224] = wde.KeyLeftControl
	keys[225] = wde.KeyLeftShift
	keys[226] = wde.KeyLeftAlt
	keys[229] = wde.KeyRightShift
	keys[230] = wde.KeyRightAlt

}
