package system

const (
	buttonA      = 0
	buttonB      = 1
	buttonSelect = 2
	buttonStart  = 3
	buttonUp     = 4
	buttonDown   = 5
	buttonLeft   = 6
	buttonRight  = 7

	strobeBit = 0
)

// Controller is an abstraction to handle joypad button input.
// It is not implemented in this package and should instead
// be implemented using the emulator's input method of choice.
type Controller interface {
	Up() bool
	Down() bool
	Left() bool
	Right() bool
	A() bool
	B() bool
	Start() bool
	Select() bool
}

type joypad struct {
	controller Controller
	strobe     bool
	input      int
}

func (j *joypad) read() uint8 {
	if !j.strobe {
		return 0
	}

	prev := j.input
	if j.input <= buttonRight {
		j.input++
	}

	var result uint8 = 0

	switch prev {
	case buttonA:
		if j.controller.A() {
			result |= 1
		}
	case buttonB:
		if j.controller.B() {
			result |= 1
		}
	case buttonUp:
		if j.controller.Up() {
			result |= 1
		}
	case buttonDown:
		if j.controller.Down() {
			result |= 1
		}
	case buttonLeft:
		if j.controller.Left() {
			result |= 1
		}
	case buttonRight:
		if j.controller.Right() {
			result |= 1
		}
	case buttonStart:
		if j.controller.Start() {
			result |= 1
		}
	case buttonSelect:
		if j.controller.Select() {
			result |= 1
		}
	default:
		result |= 1
	}
	return result
}

func (j *joypad) write(v uint8) {
	if isBitSet(v, strobeBit) {
		j.strobe = false
	} else {
		j.input = 0
		j.strobe = true
	}
}
