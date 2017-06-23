package msg

type Action uint32 // "int" in go-gl docs
const (
	Release Action = 0
	Press   Action = 1
	Repeat  Action = 2 // fired at regular intervals while key is held down
)

/*


original versions


*/

const (
	GLFW_MOUSE_BUTTON_1      = 0
	GLFW_MOUSE_BUTTON_2      = 1
	GLFW_MOUSE_BUTTON_3      = 2
	GLFW_MOUSE_BUTTON_4      = 3
	GLFW_MOUSE_BUTTON_5      = 4
	GLFW_MOUSE_BUTTON_6      = 5
	GLFW_MOUSE_BUTTON_7      = 6
	GLFW_MOUSE_BUTTON_8      = 7
	GLFW_MOUSE_BUTTON_LAST   = GLFW_MOUSE_BUTTON_8
	GLFW_MOUSE_BUTTON_LEFT   = GLFW_MOUSE_BUTTON_1
	GLFW_MOUSE_BUTTON_RIGHT  = GLFW_MOUSE_BUTTON_2
	GLFW_MOUSE_BUTTON_MIDDLE = GLFW_MOUSE_BUTTON_3
)

/*


go-gl versions


*/

type MouseButton uint32 // "int" in go-gl docs
const (
	MouseButton1      MouseButton = GLFW_MOUSE_BUTTON_1
	MouseButton2      MouseButton = GLFW_MOUSE_BUTTON_2
	MouseButton3      MouseButton = GLFW_MOUSE_BUTTON_3
	MouseButton4      MouseButton = GLFW_MOUSE_BUTTON_4
	MouseButton5      MouseButton = GLFW_MOUSE_BUTTON_5
	MouseButton6      MouseButton = GLFW_MOUSE_BUTTON_6
	MouseButton7      MouseButton = GLFW_MOUSE_BUTTON_7
	MouseButton8      MouseButton = GLFW_MOUSE_BUTTON_8
	MouseButtonLast   MouseButton = GLFW_MOUSE_BUTTON_LAST
	MouseButtonLeft   MouseButton = GLFW_MOUSE_BUTTON_LEFT
	MouseButtonRight  MouseButton = GLFW_MOUSE_BUTTON_RIGHT
	MouseButtonMiddle MouseButton = GLFW_MOUSE_BUTTON_MIDDLE
)
