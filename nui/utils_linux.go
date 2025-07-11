package nui

import "github.com/u00io/nui/nuikey"

var linuxKeyToNuiKey = map[int]nuikey.Key{
	0x09: nuikey.KeyEsc,
	0x0A: nuikey.Key1,
	0x0B: nuikey.Key2,
	0x0C: nuikey.Key3,
	0x0D: nuikey.Key4,
	0x0E: nuikey.Key5,
	0x0F: nuikey.Key6,
	0x10: nuikey.Key7,
	0x11: nuikey.Key8,
	0x12: nuikey.Key9,
	0x13: nuikey.Key0,
	0x14: nuikey.KeyMinus,
	0x15: nuikey.KeyEqual,
	0x16: nuikey.KeyBackspace,
	0x17: nuikey.KeyTab,
	0x18: nuikey.KeyQ,
	0x19: nuikey.KeyW,
	0x1A: nuikey.KeyE,
	0x1B: nuikey.KeyR,
	0x1C: nuikey.KeyT,
	0x1D: nuikey.KeyY,
	0x1E: nuikey.KeyU,
	0x1F: nuikey.KeyI,
	0x20: nuikey.KeyO,
	0x21: nuikey.KeyP,
	0x22: nuikey.KeyLeftBracket,
	0x23: nuikey.KeyRightBracket,
	0x24: nuikey.KeyEnter,
	0x25: nuikey.KeyCtrl,
	0x26: nuikey.KeyA,
	0x27: nuikey.KeyS,
	0x28: nuikey.KeyD,
	0x29: nuikey.KeyF,
	0x2A: nuikey.KeyG,
	0x2B: nuikey.KeyH,
	0x2C: nuikey.KeyJ,
	0x2D: nuikey.KeyK,
	0x2E: nuikey.KeyL,
	0x2F: nuikey.KeySemicolon,
	0x30: nuikey.KeyApostrophe,
	0x31: nuikey.KeyGrave,
	0x32: nuikey.KeyShift,
	0x33: nuikey.KeyBackslash,
	0x34: nuikey.KeyZ,
	0x35: nuikey.KeyX,
	0x36: nuikey.KeyC,
	0x37: nuikey.KeyV,
	0x38: nuikey.KeyB,
	0x39: nuikey.KeyN,
	0x3A: nuikey.KeyM,
	0x3B: nuikey.KeyComma,
	0x3C: nuikey.KeyDot,
	0x3D: nuikey.KeySlash,
	0x3E: nuikey.KeyShift,
	0x3F: nuikey.KeyNumpadAsterisk,
	0x40: nuikey.KeyAlt,
	0x41: nuikey.KeySpace,
	0x42: nuikey.KeyCapsLock,
	0x43: nuikey.KeyF1,
	0x44: nuikey.KeyF2,
	0x45: nuikey.KeyF3,
	0x46: nuikey.KeyF4,
	0x47: nuikey.KeyF5,
	0x48: nuikey.KeyF6,
	0x49: nuikey.KeyF7,
	0x4A: nuikey.KeyF8,
	0x4B: nuikey.KeyF9,
	0x4C: nuikey.KeyF10,
	0x4D: nuikey.KeyNumLock,
	0x4E: nuikey.KeyScrollLock,
	0x4F: nuikey.KeyNumpad7,
	0x50: nuikey.KeyNumpad8,
	0x51: nuikey.KeyNumpad9,
	0x52: nuikey.KeyNumpadMinus,
	0x53: nuikey.KeyNumpad4,
	0x54: nuikey.KeyNumpad5,
	0x55: nuikey.KeyNumpad6,
	0x56: nuikey.KeyNumpadPlus,
	0x57: nuikey.KeyNumpad1,
	0x58: nuikey.KeyNumpad2,
	0x59: nuikey.KeyNumpad3,
	0x5A: nuikey.KeyNumpad0,
	0x5B: nuikey.KeyNumpadDot,
	0x5F: nuikey.KeyF11,
	0x60: nuikey.KeyF12,

	0x68: nuikey.KeyEnter,
	0x69: nuikey.KeyCtrl,
	0x6A: nuikey.KeyNumpadSlash,
	0x6B: nuikey.KeyPrintScreen,
	0x6C: nuikey.KeyAlt,
	0x6E: nuikey.KeyHome,
	0x6F: nuikey.KeyArrowUp,
	0x70: nuikey.KeyPageUp,
	0x71: nuikey.KeyArrowLeft,
	0x72: nuikey.KeyArrowRight,
	0x73: nuikey.KeyEnd,
	0x74: nuikey.KeyArrowDown,
	0x75: nuikey.KeyPageDown,
	0x76: nuikey.KeyInsert,
	0x77: nuikey.KeyDelete,

	0x7F: nuikey.KeyPauseBreak,

	0x85: nuikey.KeyWin,

	0x87: nuikey.KeyContextMenu,
}

func ConvertLinuxKeyToNuiKey(macosKey int) nuikey.Key {
	if key, ok := linuxKeyToNuiKey[macosKey]; ok {
		return key
	}
	return nuikey.Key(0)
}
