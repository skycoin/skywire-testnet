package msg

/*


original versions


*/

const (
	GLFW_MOD_SHIFT   = 0x0001 // 	If this bit is set one or more Shift keys were held
	GLFW_MOD_CONTROL = 0x0002 // ^
	GLFW_MOD_ALT     = 0x0004 // ^
	GLFW_MOD_SUPER   = 0x0008 // ^
)

/*


go-gl versions


*/

type ModifierKey uint32 // "int" in go-gl docs
const (
	ModShift   ModifierKey = GLFW_MOD_SHIFT
	ModControl ModifierKey = GLFW_MOD_CONTROL
	ModAlt     ModifierKey = GLFW_MOD_ALT
	ModSuper   ModifierKey = GLFW_MOD_SUPER
)

/*


original versions


*/

const (
	GLFW_KEY_UNKNOWN    = -1
	GLFW_KEY_SPACE      = 32
	GLFW_KEY_APOSTROPHE = 39 /* ' */
	GLFW_KEY_COMMA      = 44 /* , */
	GLFW_KEY_MINUS      = 45 /* - */
	GLFW_KEY_PERIOD     = 46 /* . */
	GLFW_KEY_SLASH      = 47 /* / */

	GLFW_KEY_0 = 48
	GLFW_KEY_1 = 49
	GLFW_KEY_2 = 50
	GLFW_KEY_3 = 51
	GLFW_KEY_4 = 52
	GLFW_KEY_5 = 53
	GLFW_KEY_6 = 54
	GLFW_KEY_7 = 55
	GLFW_KEY_8 = 56
	GLFW_KEY_9 = 57

	GLFW_KEY_SEMICOLON = 59 /* ; */
	GLFW_KEY_EQUAL     = 61 /* = */

	GLFW_KEY_A = 65
	GLFW_KEY_B = 66
	GLFW_KEY_C = 67
	GLFW_KEY_D = 68
	GLFW_KEY_E = 69
	GLFW_KEY_F = 70
	GLFW_KEY_G = 71
	GLFW_KEY_H = 72
	GLFW_KEY_I = 73
	GLFW_KEY_J = 74
	GLFW_KEY_K = 75
	GLFW_KEY_L = 76
	GLFW_KEY_M = 77
	GLFW_KEY_N = 78
	GLFW_KEY_O = 79
	GLFW_KEY_P = 80
	GLFW_KEY_Q = 81
	GLFW_KEY_R = 82
	GLFW_KEY_S = 83
	GLFW_KEY_T = 84
	GLFW_KEY_U = 85
	GLFW_KEY_V = 86
	GLFW_KEY_W = 87
	GLFW_KEY_X = 88
	GLFW_KEY_Y = 89
	GLFW_KEY_Z = 90

	GLFW_KEY_LEFT_BRACKET  = 91  /* [ */
	GLFW_KEY_BACKSLASH     = 92  /* \ */
	GLFW_KEY_RIGHT_BRACKET = 93  /* ] */
	GLFW_KEY_GRAVE_ACCENT  = 96  /* ` */
	GLFW_KEY_WORLD_1       = 161 /* non-US #1 */
	GLFW_KEY_WORLD_2       = 162 /* non-US #2 */

	GLFW_KEY_ESCAPE    = 256
	GLFW_KEY_ENTER     = 257
	GLFW_KEY_TAB       = 258
	GLFW_KEY_BACKSPACE = 259
	GLFW_KEY_INSERT    = 260
	GLFW_KEY_DELETE    = 261

	GLFW_KEY_RIGHT = 262
	GLFW_KEY_LEFT  = 263
	GLFW_KEY_DOWN  = 264
	GLFW_KEY_UP    = 265

	GLFW_KEY_PAGE_UP   = 266
	GLFW_KEY_PAGE_DOWN = 267
	GLFW_KEY_HOME      = 268
	GLFW_KEY_END       = 269

	GLFW_KEY_CAPS_LOCK    = 280
	GLFW_KEY_SCROLL_LOCK  = 281
	GLFW_KEY_NUM_LOCK     = 282
	GLFW_KEY_PRINT_SCREEN = 283
	GLFW_KEY_PAUSE        = 284

	GLFW_KEY_F1  = 290
	GLFW_KEY_F2  = 291
	GLFW_KEY_F3  = 292
	GLFW_KEY_F4  = 293
	GLFW_KEY_F5  = 294
	GLFW_KEY_F6  = 295
	GLFW_KEY_F7  = 296
	GLFW_KEY_F8  = 297
	GLFW_KEY_F9  = 298
	GLFW_KEY_F10 = 299
	GLFW_KEY_F11 = 300
	GLFW_KEY_F12 = 301
	GLFW_KEY_F13 = 302
	GLFW_KEY_F14 = 303
	GLFW_KEY_F15 = 304
	GLFW_KEY_F16 = 305
	GLFW_KEY_F17 = 306
	GLFW_KEY_F18 = 307
	GLFW_KEY_F19 = 308
	GLFW_KEY_F20 = 309
	GLFW_KEY_F21 = 310
	GLFW_KEY_F22 = 311
	GLFW_KEY_F23 = 312
	GLFW_KEY_F24 = 313
	GLFW_KEY_F25 = 314

	GLFW_KEY_KP_0 = 320
	GLFW_KEY_KP_1 = 321
	GLFW_KEY_KP_2 = 322
	GLFW_KEY_KP_3 = 323
	GLFW_KEY_KP_4 = 324
	GLFW_KEY_KP_5 = 325
	GLFW_KEY_KP_6 = 326
	GLFW_KEY_KP_7 = 327
	GLFW_KEY_KP_8 = 328
	GLFW_KEY_KP_9 = 329

	GLFW_KEY_KP_DECIMAL  = 330
	GLFW_KEY_KP_DIVIDE   = 331
	GLFW_KEY_KP_MULTIPLY = 332
	GLFW_KEY_KP_SUBTRACT = 333
	GLFW_KEY_KP_ADD      = 334
	GLFW_KEY_KP_ENTER    = 335
	GLFW_KEY_KP_EQUAL    = 336

	GLFW_KEY_LEFT_SHIFT   = 340
	GLFW_KEY_LEFT_CONTROL = 341
	GLFW_KEY_LEFT_ALT     = 342
	GLFW_KEY_LEFT_SUPER   = 343

	GLFW_KEY_RIGHT_SHIFT   = 344
	GLFW_KEY_RIGHT_CONTROL = 345
	GLFW_KEY_RIGHT_ALT     = 346
	GLFW_KEY_RIGHT_SUPER   = 347

	GLFW_KEY_MENU = 348
	GLFW_KEY_LAST = GLFW_KEY_MENU
)

/*


go-gl versions


*/

const (
	KeyUnknown      = GLFW_KEY_UNKNOWN
	KeySpace        = GLFW_KEY_SPACE
	KeyApostrophe   = GLFW_KEY_APOSTROPHE
	KeyComma        = GLFW_KEY_COMMA
	KeyMinus        = GLFW_KEY_MINUS
	KeyPeriod       = GLFW_KEY_PERIOD
	KeySlash        = GLFW_KEY_SLASH
	Key0            = GLFW_KEY_0
	Key1            = GLFW_KEY_1
	Key2            = GLFW_KEY_2
	Key3            = GLFW_KEY_3
	Key4            = GLFW_KEY_4
	Key5            = GLFW_KEY_5
	Key6            = GLFW_KEY_6
	Key7            = GLFW_KEY_7
	Key8            = GLFW_KEY_8
	Key9            = GLFW_KEY_9
	KeySemicolon    = GLFW_KEY_SEMICOLON
	KeyEqual        = GLFW_KEY_EQUAL
	KeyA            = GLFW_KEY_A
	KeyB            = GLFW_KEY_B
	KeyC            = GLFW_KEY_C
	KeyD            = GLFW_KEY_D
	KeyE            = GLFW_KEY_E
	KeyF            = GLFW_KEY_F
	KeyG            = GLFW_KEY_G
	KeyH            = GLFW_KEY_H
	KeyI            = GLFW_KEY_I
	KeyJ            = GLFW_KEY_J
	KeyK            = GLFW_KEY_K
	KeyL            = GLFW_KEY_L
	KeyM            = GLFW_KEY_M
	KeyN            = GLFW_KEY_N
	KeyO            = GLFW_KEY_O
	KeyP            = GLFW_KEY_P
	KeyQ            = GLFW_KEY_Q
	KeyR            = GLFW_KEY_R
	KeyS            = GLFW_KEY_S
	KeyT            = GLFW_KEY_T
	KeyU            = GLFW_KEY_U
	KeyV            = GLFW_KEY_V
	KeyW            = GLFW_KEY_W
	KeyX            = GLFW_KEY_X
	KeyY            = GLFW_KEY_Y
	KeyZ            = GLFW_KEY_Z
	KeyLeftBracket  = GLFW_KEY_LEFT_BRACKET
	KeyBackslash    = GLFW_KEY_BACKSLASH
	KeyRightBracket = GLFW_KEY_RIGHT_BRACKET
	KeyGraveAccent  = GLFW_KEY_GRAVE_ACCENT
	KeyWorld1       = GLFW_KEY_WORLD_1
	KeyWorld2       = GLFW_KEY_WORLD_2
	KeyEscape       = GLFW_KEY_ESCAPE
	KeyEnter        = GLFW_KEY_ENTER
	KeyTab          = GLFW_KEY_TAB
	KeyBackspace    = GLFW_KEY_BACKSPACE
	KeyInsert       = GLFW_KEY_INSERT
	KeyDelete       = GLFW_KEY_DELETE
	KeyRight        = GLFW_KEY_RIGHT
	KeyLeft         = GLFW_KEY_LEFT
	KeyDown         = GLFW_KEY_DOWN
	KeyUp           = GLFW_KEY_UP
	KeyPageUp       = GLFW_KEY_PAGE_UP
	KeyPageDown     = GLFW_KEY_PAGE_DOWN
	KeyHome         = GLFW_KEY_HOME
	KeyEnd          = GLFW_KEY_END
	KeyCapsLock     = GLFW_KEY_CAPS_LOCK
	KeyScrollLock   = GLFW_KEY_SCROLL_LOCK
	KeyNumLock      = GLFW_KEY_NUM_LOCK
	KeyPrintScreen  = GLFW_KEY_PRINT_SCREEN
	KeyPause        = GLFW_KEY_PAUSE
	KeyF1           = GLFW_KEY_F1
	KeyF2           = GLFW_KEY_F2
	KeyF3           = GLFW_KEY_F3
	KeyF4           = GLFW_KEY_F4
	KeyF5           = GLFW_KEY_F5
	KeyF6           = GLFW_KEY_F6
	KeyF7           = GLFW_KEY_F7
	KeyF8           = GLFW_KEY_F8
	KeyF9           = GLFW_KEY_F9
	KeyF10          = GLFW_KEY_F10
	KeyF11          = GLFW_KEY_F11
	KeyF12          = GLFW_KEY_F12
	KeyF13          = GLFW_KEY_F13
	KeyF14          = GLFW_KEY_F14
	KeyF15          = GLFW_KEY_F15
	KeyF16          = GLFW_KEY_F16
	KeyF17          = GLFW_KEY_F17
	KeyF18          = GLFW_KEY_F18
	KeyF19          = GLFW_KEY_F19
	KeyF20          = GLFW_KEY_F20
	KeyF21          = GLFW_KEY_F21
	KeyF22          = GLFW_KEY_F22
	KeyF23          = GLFW_KEY_F23
	KeyF24          = GLFW_KEY_F24
	KeyF25          = GLFW_KEY_F25
	KeyKP0          = GLFW_KEY_KP_0
	KeyKP1          = GLFW_KEY_KP_1
	KeyKP2          = GLFW_KEY_KP_2
	KeyKP3          = GLFW_KEY_KP_3
	KeyKP4          = GLFW_KEY_KP_4
	KeyKP5          = GLFW_KEY_KP_5
	KeyKP6          = GLFW_KEY_KP_6
	KeyKP7          = GLFW_KEY_KP_7
	KeyKP8          = GLFW_KEY_KP_8
	KeyKP9          = GLFW_KEY_KP_9
	KeyKPDecimal    = GLFW_KEY_KP_DECIMAL
	KeyKPDivide     = GLFW_KEY_KP_DIVIDE
	KeyKPMultiply   = GLFW_KEY_KP_MULTIPLY
	KeyKPSubtract   = GLFW_KEY_KP_SUBTRACT
	KeyKPAdd        = GLFW_KEY_KP_ADD
	KeyKPEnter      = GLFW_KEY_KP_ENTER
	KeyKPEqual      = GLFW_KEY_KP_EQUAL
	KeyLeftShift    = GLFW_KEY_LEFT_SHIFT
	KeyLeftControl  = GLFW_KEY_LEFT_CONTROL
	KeyLeftAlt      = GLFW_KEY_LEFT_ALT
	KeyLeftSuper    = GLFW_KEY_LEFT_SUPER
	KeyRightShift   = GLFW_KEY_RIGHT_SHIFT
	KeyRightControl = GLFW_KEY_RIGHT_CONTROL
	KeyRightAlt     = GLFW_KEY_RIGHT_ALT
	KeyRightSuper   = GLFW_KEY_RIGHT_SUPER
	KeyMenu         = GLFW_KEY_MENU
	KeyLast         = GLFW_KEY_LAST
)

/*

escape sequence chars

*/

const (
	EscBackSlash      = 92
	EscNewLine        = 10
	EscCarriageReturn = 13
	EscTab            = 9
	EscBackSpace      = 8
)
